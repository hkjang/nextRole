package connectors

import "time"

// Presets returns disabled, credential-free starter configurations. The admin
// supplies an approved Work24 API key and tests the actual response before sync.
// Endpoint/field contracts were checked against work24.go.kr on 2026-09-27.
func Presets() []Config { return Work24Presets(time.Now()) }

func Work24Presets(now time.Time) []Config {
	return []Config{
		{
			Name: "고용24 채용정보", Type: "xml", Dataset: "jobs", Method: "GET",
			Endpoint:    "https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo210L01.do",
			APIKeyParam: "authKey", RootPath: "wantedRoot.wanted",
			Params: map[string]string{"callTp": "L", "returnType": "XML", "startPage": "1", "display": "100"},
			Mapping: map[string]string{
				"id": "wantedAuthNo", "title": "title", "organization": "company",
				"region": "region", "url": "wantedInfoUrl", "salary": "sal", "deadline": "closeDt",
				"description": "title", "skills": "=[]", "source.name": "=고용24 채용정보", "source.synthetic": "=false",
			},
		},
		{
			Name: "고용24 국민내일배움카드 훈련", Type: "xml", Dataset: "training", Method: "GET",
			Endpoint:    "https://www.work24.go.kr/cm/openApi/call/hr/callOpenApiSvcInfo310L01.do",
			APIKeyParam: "authKey", RootPath: "HRDNet.srchList.scn_list",
			Params: map[string]string{
				"returnType": "XML", "outType": "1", "pageNum": "1", "pageSize": "100",
				"srchTraStDt": now.Format("20060102"), "srchTraEndDt": now.AddDate(0, 3, 0).Format("20060102"),
				"sort": "ASC", "sortCol": "2",
			},
			Mapping: map[string]string{
				"id": "concat:trprId,trprDegr", "title": "title", "organization": "subTitle", "region": "address",
				"url": "titleLink", "description": "contents", "skills": "=[]",
				"source.name": "=고용24 국민내일배움카드", "source.synthetic": "=false",
			},
		},
	}
}
