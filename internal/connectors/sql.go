package connectors

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// ValidateQuery is a conservative guard, not a SQL sandbox. Connector accounts
// must have SELECT-only access to dedicated views, without privileged function
// execution. Both adapters additionally start a read-only transaction and impose
// execution deadlines. Expressions, subqueries and read-only CTEs are supported.
func ValidateQuery(query string) error {
	if len(query) == 0 || len(query) > 50000 {
		return errors.New("조회 SQL은 1~50,000자로 입력하세요")
	}
	tokens, err := sqlTokens(query)
	if err != nil {
		return err
	}
	if len(tokens) == 0 || tokens[0] != "SELECT" && tokens[0] != "WITH" {
		return errors.New("읽기 전용 SELECT 또는 WITH 조회만 허용합니다")
	}
	blocked := map[string]bool{}
	for _, word := range strings.Fields("INSERT UPDATE DELETE MERGE REPLACE UPSERT DROP ALTER CREATE TRUNCATE COPY CALL EXEC EXECUTE DO GRANT REVOKE VACUUM ANALYZE REFRESH LOCK UNLOCK SET RESET BEGIN START COMMIT ROLLBACK PREPARE DEALLOCATE LISTEN NOTIFY UNLISTEN LOAD ATTACH DETACH INTO OUTFILE DUMPFILE USE SHUTDOWN KILL FLUSH INSTALL UNINSTALL HANDLER IMPORT EXPORT LOAD_FILE SLEEP BENCHMARK GET_LOCK RELEASE_LOCK RELEASE_ALL_LOCKS PG_SLEEP PG_SLEEP_FOR PG_SLEEP_UNTIL PG_READ_FILE PG_READ_BINARY_FILE PG_WRITE_FILE PG_LS_DIR PG_LS_LOGDIR PG_LS_WALDIR PG_STAT_FILE LO_IMPORT LO_EXPORT DBLINK DBLINK_EXEC DBLINK_CONNECT DBLINK_CONNECT_U PG_TERMINATE_BACKEND PG_CANCEL_BACKEND PG_RELOAD_CONF PG_ROTATE_LOGFILE PG_ADVISORY_LOCK PG_ADVISORY_XACT_LOCK SET_CONFIG NEXTVAL SETVAL") {
		blocked[word] = true
	}
	for _, token := range tokens {
		if blocked[token] || strings.HasPrefix(token, "DBLINK_") || strings.HasPrefix(token, "PG_ADVISORY_") {
			return errors.New("SQL에 허용되지 않은 명령 또는 함수가 포함되어 있습니다")
		}
	}
	return nil
}

// sqlTokens rejects comments (including executable MySQL comments), positional
// dollar quoting, escapes that depend on SQL mode, and multiple statements. It
// ignores string contents while inspecting quoted identifiers as well.
func sqlTokens(query string) ([]string, error) {
	runes := []rune(strings.TrimSpace(query))
	if len(runes) > 0 && runes[len(runes)-1] == ';' {
		runes = runes[:len(runes)-1]
	}
	var tokens []string
	for i := 0; i < len(runes); {
		r := runes[i]
		if r == ';' || r == '#' || r == '$' || r == '\\' || r == '-' && i+1 < len(runes) && runes[i+1] == '-' || r == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			return nil, errors.New("SQL 주석, 다중 명령, 달러 인용 및 역슬래시 이스케이프는 허용하지 않습니다")
		}
		if r == '\'' || r == '"' || r == '`' {
			quote := r
			i++
			var value strings.Builder
			closed := false
			for i < len(runes) {
				if runes[i] == '\\' {
					return nil, errors.New("SQL 문자열의 역슬래시 이스케이프는 허용하지 않습니다")
				}
				if runes[i] == quote {
					if i+1 < len(runes) && runes[i+1] == quote {
						value.WriteRune(quote)
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				value.WriteRune(runes[i])
				i++
			}
			if !closed {
				return nil, errors.New("SQL 인용부호가 닫히지 않았습니다")
			}
			if quote != '\'' {
				tokens = append(tokens, strings.ToUpper(value.String()))
			}
			continue
		}
		if unicode.IsLetter(r) || r == '_' {
			start := i
			i++
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			tokens = append(tokens, strings.ToUpper(string(runes[start:i])))
			continue
		}
		i++
	}
	return tokens, nil
}

func fetchSQL(ctx context.Context, c Config) ([]map[string]any, error) {
	var db *sql.DB
	if c.Type == "postgres" {
		config, err := pgx.ParseConfig(c.DSN)
		if err != nil {
			return nil, errors.New("PostgreSQL 연결 문자열 형식이 올바르지 않습니다")
		}
		config.ConnectTimeout = 5 * time.Second
		config.RuntimeParams["statement_timeout"] = "15000"
		config.RuntimeParams["default_transaction_read_only"] = "on"
		config.RuntimeParams["application_name"] = "NextRole connector"
		db = stdlib.OpenDB(*config)
	} else {
		config, err := mysql.ParseDSN(c.DSN)
		if err != nil {
			return nil, errors.New("MySQL 연결 문자열 형식이 올바르지 않습니다")
		}
		config.MultiStatements = false
		config.AllowAllFiles = false
		config.Timeout = 5 * time.Second
		config.ReadTimeout, config.WriteTimeout = FetchTimeout, FetchTimeout
		if config.Params == nil {
			config.Params = map[string]string{}
		}
		config.Params["max_execution_time"] = "15000"
		connector, err := mysql.NewConnector(config)
		if err != nil {
			return nil, errors.New("MySQL 연결 설정을 적용할 수 없습니다")
		}
		db = sql.OpenDB(connector)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.New("읽기 전용 DB 연결에 실패했습니다. 연결 설정과 SELECT 권한을 확인하세요")
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, c.Query)
	if err != nil {
		return nil, errors.New("DB 조회에 실패했습니다. 읽기 권한, SQL 문법과 15초 실행 제한을 확인하세요")
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil || len(columns) > 200 {
		return nil, errors.New("DB 결과 열은 200개까지 지원합니다")
	}
	seen := map[string]bool{}
	for _, column := range columns {
		if column == "" || seen[column] {
			return nil, errors.New("DB 결과 열 이름이 비어 있거나 중복됩니다. SQL 별칭을 지정하세요")
		}
		seen[column] = true
	}
	result := []map[string]any{}
	var totalBytes int
	for rows.Next() {
		if len(result) >= MaxRecords {
			return nil, errors.New("한 번에 5,000개 레코드까지만 가져올 수 있습니다. SQL LIMIT를 설정하세요")
		}
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, errors.New("DB 결과를 읽을 수 없습니다")
		}
		record := make(map[string]any, len(columns))
		for i, value := range values {
			switch v := value.(type) {
			case []byte:
				// PostgreSQL JSON and MySQL JSON/text columns may arrive as []byte.
				var structured any
				if json.Valid(v) && (strings.HasPrefix(strings.TrimSpace(string(v)), "{") || strings.HasPrefix(strings.TrimSpace(string(v)), "[")) && json.Unmarshal(v, &structured) == nil {
					record[columns[i]] = structured
				} else {
					record[columns[i]] = string(v)
				}
			case time.Time:
				record[columns[i]] = v.Format(time.RFC3339)
			default:
				record[columns[i]] = value
			}
		}
		encoded, err := json.Marshal(record)
		if err != nil {
			return nil, errors.New("DB 결과에 지원하지 않는 값 형식이 있습니다")
		}
		totalBytes += len(encoded)
		if totalBytes > MaxResponseBytes {
			return nil, errors.New("DB 조회 결과가 10 MiB 제한을 초과했습니다. 조회 범위를 줄이세요")
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("DB 결과 읽기에 실패했거나 실행 제한 시간을 초과했습니다")
	}
	// Rollback intentionally closes a read-only transaction, even on success.
	return result, nil
}
