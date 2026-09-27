# 고용24 데이터 활용 및 출처 관리

**NextRole v1.1.0**

NextRole은 한국고용정보원이 고용24를 통해 제공하는 공개 API 4종을 활용합니다. 아래 정상 응답 항목은 공식 명세를 기준으로 정리했습니다. 전체 응답을 보관하지 않고 관리자가 활용 목적에 맞게 선택한 필드만 저장합니다.

공공 API 원천, 이용자 입력, 자체 합성, 자체 가공 데이터는 서로 구분합니다. 내부 역량 수준·가중치·적합도는 고용24 또는 NCS의 공식 평가값이 아닙니다.

## 1. 직업정보 상세 — 요약

- 제공기관: 한국고용정보원·고용24
- [공식 명세](https://www.work24.go.kr/cm/e/a/0110/selectOpenApiSvcInfo.do?fullApiSvcId=000000000000000000000000000093%5E000000000000000000000000000090)
- 요청: `https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo212D01.do`
- HTTP GET / XML: `authKey`, `returnType=XML`, `target=JOBDTL`, `jobGb=1`, `jobCd`, `dtlGb=1`
- 응답 루트: `jobSum`. 그 안의 같은 이름 `jobSum` 필드는 하는 일입니다.

정상 응답 전체 항목: 직업코드(jobCd), 직업 대분류명(jobLrclNm), 직업 중분류명(jobMdclNm), 직업 소분류명(jobSmclNm), 하는 일(jobSum), 되는 길(way), 관련전공 목록(relMajorList: majorCd·majorNm), 관련자격증 목록(relCertList: certNm), 임금(sal), 직업만족도(jobSatis), 일자리전망(jobProspect), 일자리현황(jobStatus), 업무수행능력(jobAbil), 지식(knowldg), 업무환경(jobEnv), 성격(jobChr), 흥미(jobIntrst), 직업가치관(jobVals), 업무활동 중요도(jobActvImprtncs), 업무활동 수준(jobActvLvls), 관련직업 목록(relJobList: jobCd·jobNm).

활용: 직업코드, 하는 일, 되는 길, 업무수행능력, 지식, 관련자격증, 일자리전망을 중심으로 목표 직무를 설명하고 역량 분석의 기초자료로 사용합니다. 원천 문자열을 내부 0~5 요구 수준으로 자동 간주하지 않습니다.

## 2. 표준직무기술서

- 제공기관: 한국고용정보원·고용24
- [공식 명세](https://www.work24.go.kr/cm/e/a/0110/selectOpenApiSvcInfo.do?apiSvcId=000000000000000000000000000087&fullApiSvcId=000000000000000000000000000096%5E000000000000000000000000000084&upprApiSvcId=000000000000000000000000000086)
- 요청: `https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo215L01.do`
- HTTP GET / JSON: `authKey`, `jobCont`, 선택 `limit`(기본 5), 공식 예제의 `returnType=JSON`
- 응답: `result` 객체 안에 능력단위명 키별 객체. 배열이라고 가정하지 않습니다.

정상 응답 전체 항목: NCS 능력단위명(job_sdvn), NCS 능력단위 정의(ablt_def), NCS 소분류명(job_scfn), NCS 소분류코드(job_scla_cd), NCS 대분류코드(job_lrcl_cd), NCS 중분류명(job_mcn), NCS 대분류명(job_lcfn), NCS 세분류코드(job_sdvn_cd), NCS 중분류코드(job_mlsf_cd), NCS 능력단위 코드(ablt_unit), 지식·기술·태도(knwg_tchn_attd).

활용: 능력단위 코드·명칭·정의, 소분류코드, 지식·기술·태도로 사용자 경력과 목표 직무를 연결합니다. 내부 요구 수준과 가중치는 별도 관리자 검토를 거치며 NCS 공식 수준과 구분합니다. 이용자의 경력을 외부 API에 자동 전송하지 않습니다. 관리자가 입력한 수행직무내용으로 조회합니다.

## 3. 채용정보 목록

- 제공기관: 한국고용정보원·고용24
- [공식 명세](https://www.work24.go.kr/cm/e/a/0110/selectOpenApiSvcInfo.do?fullApiSvcId=000000000000000000000000000000)
- 요청: `https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo210L01.do`
- HTTP GET / XML: `authKey`, `callTp=L`, `returnType=XML`, `startPage`, `display`; 선택 `occupation`, `region`, `keyword` 등
- 루트: `wantedRoot.wanted`. `startPage` 최대 1000, `display` 최대 100.

정상 응답 전체 항목: 구인인증번호(wantedAuthNo), 회사명(company), 사업자등록번호(busino), 업종(indTpNm), 채용제목(title), 임금형태(salTpNm), 급여(sal), 최소임금액(minSal), 최대임금액(maxSal), 근무지역(region), 근무형태(holidayTpNm), 최소학력(minEdubg), 최대학력(maxEdubg), 경력(career), 등록일자(regDt), 마감일자(closeDt), 정보제공처(infoSvc), 채용정보 URL(wantedInfoUrl), 모바일 채용정보 URL(wantedMobileInfoUrl), 근무지 우편주소(zipCd), 근무지 도로명주소(strtnmCd), 근무지 기본주소(basicAddr), 근무지 상세주소(detailAddr), 고용형태코드(empTpCd), 직종코드(jobsCd), 최종수정일(smodifyDtm). 페이지 메타: 총건수(total), 시작페이지(startPage), 출력건수(display).

활용: 공고 식별번호, 제목, 기업명, 직종, 지역, 급여, 마감일, 원문 URL. 이 목록 API에는 요구기술 목록이 없습니다. 제목에서 추출한 기술을 API 제공 요구기술처럼 표시하거나 임의 생성하지 않습니다. 실제 채용수요 집계에서 합성 자료는 제외합니다.

## 4. 국민내일배움카드 훈련과정 목록

- 제공기관: 한국고용정보원·고용24
- [공식 명세](https://www.work24.go.kr/cm/e/a/0110/selectOpenApiSvcInfo.do?fullApiSvcId=000000000000000000000000000004)
- 요청: `https://www.work24.go.kr/cm/openApi/call/hr/callOpenApiSvcInfo310L01.do`
- HTTP GET: XML·JSON 지원, 초기 연동 XML. `authKey`, `returnType=XML`, `outType=1`, `pageNum`, `pageSize`, `srchTraStDt`, `srchTraEndDt`, `sort`, `sortCol`; 선택 지역·NCS 조건
- 루트: `HRDNet.srchList.scn_list`. `pageNum` 최대 1000, `pageSize` 최대 100.
- `srchTraStDt`와 `srchTraEndDt`는 **훈련 시작일의 검색 범위**입니다.

정상 응답 전체 항목: 주소(address), 자격증(certificate), 콘텐츠(contents), 수강비(courseMan), 고용보험 3개월 취업인원 수(eiEmplCnt3), 고용보험 3개월 취업누적인원 10인 이하 여부(eiEmplCnt3Gt10), 고용보험 3개월 취업률(eiEmplRate3), 고용보험 6개월 취업률(eiEmplRate6), 등급(grade), 훈련기관 코드(instCd), NCS 코드(ncsCd), 실제 훈련비(realMan), 수강신청 인원(regCourseMan), 만족도 점수(stdgScor), 부제목(subTitle), 부제목 링크(subTitleLink), 전화번호(telNo), 제목(title), 제목 아이콘(titleIcon), 제목 링크(titleLink), 훈련종료일자(traEndDate), 훈련시작일자(traStartDate), 훈련대상(trainTarget), 훈련구분(trainTargetCd), 훈련기관 ID(trainstCstId), 지역 중분류코드(trngAreaCd), 훈련과정 순차(trprDegr), 훈련과정 ID(trprId), 주말·주중 구분(wkendSe), 정원(yardMan). 페이지 메타: 총건수(scn_cnt), 현재 페이지(pageNum), 페이지당 출력건수(pageSize).

`eiEmplCnt3Gt10`은 제공 중단으로 Null을 반환하는 호환 필드이므로 활용·저장하지 않습니다. 과정 ID와 회차를 함께 식별자로 사용합니다. 내용·NCS·주소·훈련 시작/종료일·원문 링크로 부족 역량과 과정을 연결합니다. 수강비/실제 훈련비는 개인의 확정 본인부담금이 아닙니다.

## 5. 이용자 입력

- 제공자: 이용자 본인. 외부 출처 URL·API 없음
- 활용 데이터: 경력 프로필·이력서
- 항목: 경력, 프로젝트, 보유 기술, 자격, 희망 직무, 희망지역, 주당 학습시간
- 목적: 역량 구조화, 적합도·역량 격차 계산, 경력전환 시뮬레이션
- 이용자의 동의를 받고 필요한 정보만 수집합니다. 분석에 불필요한 식별정보는 제외합니다.

## 6. 자체 합성 데이터

- 제공자: NextRole 자체 생성. 외부 출처 URL·API 없음
- 활용 데이터: 시연·검증용 합성 경력·직무·훈련·채용 자료
- 항목: 가상 식별번호, 설명, 역량, 요구수준, 출처, 합성 여부
- 목적: 외부 API가 없는 환경에서 기능 검증·시연
- 실제 개인정보를 포함하지 않으며 합성임을 표시하고 실제 채용수요 집계에서 제외합니다.

## 7. 자체 가공 데이터

- 제공자: NextRole. 고용24 원천자료를 바탕으로 별도 가공
- 활용 데이터: 직무·역량 매핑 사전
- 출처: 연결된 1~4번 공식 출처 URL. 별도 외부 제공 API 없음
- 항목: 원천 직무코드, 내부 역량 식별번호, 역량 별칭, 요구수준, 가중치, 매핑 근거, 버전
- 목적: 표현 표준화, 격차 계산, What-if 적합도 변화·전환 경로 비교
- 원천자료와 별도 저장하며 관리자가 근거·수준·가중치를 검토한 활성 버전만 계산에 사용합니다.

## 연동 검증 상태

공공데이터 이용 신청·승인을 거쳐 발급받은 인증키가 필요합니다. 공식 명세와 합성 정상·오류 응답을 이용해 구현을 검증합니다. 실제 승인 인증키가 제공되지 않은 상태에서 실데이터 조회 성공을 주장하지 않습니다. 관리자는 연결 테스트 결과와 실제 동기화 기록을 확인한 후 운영에 사용해야 합니다.


## 구현과 운영 절차

1. 관리자 **데이터 연동**에서 네 프리셋 중 하나를 선택하고 승인 인증키·조회 조건·저장할 필드를 지정합니다. 필수 식별·표시 항목은 유지합니다.
2. 연결 테스트는 한 페이지의 응답을 검사하고, 동기화는 선택 필드만 원천 레코드에 저장합니다. 전체 API 응답을 통째로 저장하거나 전체 페이지를 자동 순회하지 않습니다.
3. **원천·가공 데이터**에서 제공기관, 공식 API 종류, 원천 ID, 수집 시각, 선택 필드와 실제 내용을 확인합니다.
4. **역량 매핑 검토**에서 원천 레코드에 근거한 내부 역량 ID·이름·별칭·요구 수준·가중치·근거를 작성하고 검토한 버전을 게시합니다. 원천 자체와 가공 매핑은 별도 저장합니다.
5. 이용자는 **개인정보·동의**에서 현재 안내를 확인한 뒤 경력을 분석합니다. 고용24 요청에 개인 경력을 자동 넣지 않습니다.

### 코드·요구 수준을 연결할 때

직업상세의 `jobCd`와 채용목록의 `jobsCd`는 같은 분류체계라고 단정하지 않습니다. 관리자가 별도로 검토한 채용 직종코드를 가공 매핑의 `jobCodes`에 기록해야 코드로 연결합니다. NCS 기반 훈련 연결도 원천의 능력단위 코드 또는 `job_sdvn_cd` 세분류코드와 과정의 `ncsCd`가 정확히 일치할 때만 사용하며 접두어를 임의로 잘라 맞추지 않습니다.

내부 역량 수준과 별칭은 서비스의 비교를 위한 설정입니다. 공식 NCS 수준·국가 자격·능력단위 동등성을 보증하지 않습니다. 요구 경력·학력 기준이 없는 가공 모델에는 해당 점수를 임의로 부여하지 않습니다. 합성 모드를 끄면 내장 합성 직무도 제외되며, 유효한 게시 매핑이나 별도 승인 직무가 없으면 목록은 비어 있습니다.

### 매핑 이력과 변경 관리

매핑은 초안 저장·수정·게시·삭제 때 버전을 남깁니다. 수정하면 초안으로 돌아가 다시 게시해야 합니다. 원문 내용이 변경되거나 원천이 삭제되면 기존 매핑을 계산에서 제외합니다. 동일 원문의 재수집 시각이나 선택 필드 나열 순서만 달라지는 경우에는 불필요한 재검토를 요구하지 않습니다. 삭제 후에도 매핑 이력은 유지합니다.

### 경력 자료 수집 안내와 철회

관리자는 안내문과 버전을 설정하며 내용 변경 시 새 버전으로 재동의를 받습니다. 이용자가 철회하면 본인의 경력·저장 결과·실행계획·선호·검토 요청을 삭제하고 계정과 개인 API 키는 유지합니다. 이력서 원본 파일을 저장하지 않으며 분석용 이름·이메일·전화번호 등은 자동 제외하지만 완전한 익명화는 아니므로 결과 확인을 안내합니다. 외부 AI를 활성화한 경우 명시적으로 실행한 AI 기능의 입력이 설정 서버로 전달될 수 있습니다. 원천 데이터 조회와 동의 API 계약은 [구현 계약](architecture.md)을 참고합니다.
