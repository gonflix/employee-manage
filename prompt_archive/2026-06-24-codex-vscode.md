# 사내 직원 관리 시스템 구축

> Go로 Web 사내 직원 관리 시스템을 개발할 것이다.
> PostgresSQL DDL을 schema.sql 파일을 생성하여 그 안에 만들고
> 웹 클라이언트와 서버 간의 Rest API를 짠 emp_swagger.yaml 파일을 만들어봐
>
> [주요 기능]
> (참고: UI는 직원용/관리자용으로 나눠져 있음)
> 1. 직원 로그인
> 2. 직원이 자신의 인적사항을 조회
> 3. 직원이 자신의 인적사항을 수정
> 4. 관리자 로그인
> 5. 관리자가 직원 계정 생성/퇴사 처리
> * 퇴사처리되면 즉시 차단
> 6. 관리자가 전체 직원 목록 조회, 개별 상세정보 조회(상세정보에는 직원의 배경 정보도 포함됨)
> * 직원의 배경정보는 Background Check API로 조회

<details><summary>7 previous messages</summary>

> 좋아요, 먼저 저장소 구조와 지침을 살짝 확인한 뒤 `schema.sql`과 `emp_swagger.yaml`을 설계해둘게요.
>
> `find`가 조금 느리네요; 우선 보이는 파일부터 확인하고 필요한 설계 파일을 바로 만들겠습니다.
>
> README에 핵심 제약이 더 있네요. 응답 포맷, 블랙리스트, 배경조회 실패 케이스까지 API 명세에 반영하겠습니다.
>
> 이제 DDL은 인증/직원/퇴사 차단 흐름 중심으로, Swagger는 UI 역할별 API 경계를 명확히 나눠서 작성하겠습니다.
>
> <details><summary>Created 2 files</summary>
>
> - Wrote `./emp_swagger.yaml` (+657 -0)
> - Wrote `./schema.sql` (+137 -0)
>
> </details>
>
> 파일 두 개가 생겼습니다. 이제 문법과 주요 경로가 깨지지 않았는지 가볍게 검증할게요.
>
> YAML 파싱은 통과했어요. 마지막으로 변경 파일의 핵심 위치만 잡아두고 마무리하겠습니다.
</details>

완료했습니다. 작은 설계 뼈대가 아니라 바로 구현에 연결될 수 있게 꽤 단단히 잡아뒀어요.

- `schema.sql:1` 생성: PostgreSQL DDL, 직원/계정/부서/퇴사 블랙리스트/배경조회 로그 포함
- `schema.sql:111` 퇴사 처리 시 계정 `TERMINATED` 변경 및 블랙리스트 자동 등록 트리거 추가
- `emp_swagger.yaml:1` 생성: OpenAPI 3.0.3 REST API 명세 작성
- `emp_swagger.yaml:22` 직원 로그인, 내 정보 조회/수정 API 포함
- `emp_swagger.yaml:100` 관리자 로그인, 직원 생성/목록/상세/퇴사 처리 API 포함
- `emp_swagger.yaml:216` Background Check API 재시도 정책과 실패 시 500 처리 명시

검증으로 `emp_swagger.yaml` YAML 파싱까지 통과했습니다.
참고로 작업 전부터 `README.md` 수정과 `.omc/` 미추적 항목이 이미 보입니다.

> [schema.sql](schema.sql) 로그성 테이블은 필요없어

<details><summary>4 previous messages</summary>

> 좋아요, 로그성 테이블은 걷어내고 핵심 운영 테이블만 남기겠습니다. DDL 참조도 같이 정리할게요.
>
> 확인해보니 로그성 테이블은 변경 로그와 배경조회 요청 이력 두 개입니다. 둘 다 제거하고 남은 인덱스도 맞춰줄게요.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./schema.sql` (+0 -22)
>
> </details>
>
> 정리 완료했습니다. 이제 로그성 이름이 남아있지 않은지만 빠르게 확인합니다.
</details>

수정 완료했습니다.

- 제거: `employee_personal_change_logs` 테이블
- 제거: `background_check_requests` 테이블
- 제거: 관련 인덱스 `idx_background_check_requests_employee_id`
- 유지: 퇴사자 즉시 차단용 `terminated_employee_blacklist`는 운영 기능 테이블이라 남겼습니다

현재 `schema.sql:7` 기준 테이블은 `departments`, `employees`, `user_accounts`, `terminated_employee_blacklist`만 남아 있습니다.

> [emp_swagger.yaml](emp_swagger.yaml)
> 퇴사처리된 직원의 요청의 "code"는 BLACKLISTED_EMPLOYEE 로 응답하되 그 외의 실패 응답에는 "code"를 보내지마.
> (1)403 (2)퇴사처리된 직원의 요청이 아닌 401: "message"를 "Authentication Failed"로,
> (3) 400:  "message"를 "Invalid Request"로,
> (4)그 외의 모든 에러: "message"는 "Internal Server Error"를 기본값으로 써.

<details><summary>9 previous messages</summary>

> 좋아요, 실패 응답 규칙을 공통 스키마 쪽에서 깔끔하게 재정렬하겠습니다. `BLACKLISTED_EMPLOYEE`만 예외로 남기는 식이 제일 덜 흔들려요.
>
> 현재는 모든 에러가 같은 `ErrorResponse`라서 의미가 섞입니다. 상태별 에러 스키마를 나눠 명세에 직접 반영하겠습니다.
>
> 여기서 401은 “일반 인증 실패”와 “퇴사자 차단”이 공존하니 `oneOf`로 표현하겠습니다. 403은 `Forbidden`, 나머지는 요청하신 기본 메시지로 맞출게요.
>
> 명세를 상태별 스키마로 분리했습니다. 이제 `code`가 에러 공통에 남아있지 않은지 확인할게요.
>
> 한 가지 더 엄밀하게 맞추겠습니다. 퇴사자 차단은 직원용 화면만이 아니라 “요청 주체” 기준이라, 모든 401에 블랙리스트 응답 가능성을 열어두겠습니다.
>
> 검증은 통과했습니다. 마지막으로 헷갈릴 수 있는 미사용 메타 스키마도 정리해둘게요.
>
> `ApiMeta`는 미사용이지만 실패 응답 규칙과 혼동될 수 있어 제거하겠습니다. 깔끔한 게 좋아요.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./emp_swagger.yaml` (+0 -17)
>
> </details>
>
> 마지막 파싱과 키워드 확인만 돌려서, 명세가 의도대로 고정됐는지 보겠습니다.
</details>

수정 완료했습니다.

- `emp_swagger.yaml:356` 실패 응답 기본 스키마에서 `code` 제거
- `emp_swagger.yaml:390` 퇴사자 차단 응답만 `code: BLACKLISTED_EMPLOYEE` 포함
- `emp_swagger.yaml:383` 일반 401 메시지 기본값을 `Authentication Failed`로 지정
- `emp_swagger.yaml:374` 400 메시지 기본값을 `Invalid Request`로 지정
- `emp_swagger.yaml:420` 기타 에러 메시지 기본값을 `Internal Server Error`로 지정
- `emp_swagger.yaml:409` 403은 별도 `ForbiddenErrorResponse`로 분리

YAML 파싱도 통과했습니다.

> 클린 아키텍처 구조를 잡아놨고 GORM, PostgresDB 설정을 해놨어.
> 웹서버는 Echo로 만들었어.
> [emp_swagger.yaml](emp_swagger.yaml) 파일을 참고하여 Handler, Service, Repository 레이어를 완성해.

<details><summary>35 previous messages</summary>

> 좋습니다, 이제 실제 코드 지형을 읽고 Swagger 명세에 맞춰 레이어를 이어붙이겠습니다. 먼저 구조와 기존 컨벤션부터 파악할게요.
>
> 구조가 작고 좋네요. 이제 각 레이어의 현재 형태를 읽고, 최소한의 확장으로 맞춰보겠습니다.
>
> 현재는 뼈대만 있는 상태라, 도메인 모델/DTO/라우팅까지 함께 세워야 합니다. Swagger 기준으로 한 번에 연결하겠습니다.
>
> Swagger와 현재 DB 스키마를 기준으로 필요한 타입이 꽤 많습니다. 도메인 계약부터 잡고 Repository/Service/Handler 순서로 붙일게요.
>
> <details><summary>Deleted a file</summary>
>
> - Deleted `./internal/domain/employee.go` (+0 -13)
>
> </details>
>
> 도메인 파일은 빈 인터페이스뿐이라 새로 구성하는 편이 안전합니다. 이어서 전체 계약을 넣겠습니다.
>
> <details><summary>Created a file</summary>
>
> - Wrote `./internal/domain/employee.go` (+163 -0)
>
> </details>
>
> 도메인 계약이 잡혔습니다. 이제 HTTP 상태 매핑에 필요한 에러 타입을 보강하겠습니다.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./internal/domain/errors.go` (+7 -2)
>
> </details>
>
> 다음은 GORM Repository입니다. 트랜잭션으로 직원 생성/퇴사처리까지 원자적으로 묶겠습니다.
>
> <details><summary>Created a file, deleted a file</summary>
>
> - Deleted `./internal/repository/employeeRepository.go` (+0 -18)
> - Wrote `./internal/repository/employeeRepository.go` (+238 -0)
>
> </details>
>
> Repository가 붙었습니다. 이제 인증 토큰, 권한 체크, 블랙리스트 캐시, Background Check 재시도 로직을 Service에 모으겠습니다.
>
> <details><summary>Deleted a file</summary>
>
> - Deleted `./internal/service/employeeService.go` (+0 -15)
>
> </details>
>
> Service가 핵심입니다. 외부 API 실패 정책까지 여기서 구현해둘게요.
>
> <details><summary>Created a file</summary>
>
> - Wrote `./internal/service/employeeService.go` (+502 -0)
>
> </details>
>
> Service까지 들어갔습니다. 이제 Echo Handler에서 요청 바인딩, 인증 헤더 파싱, 응답 포맷을 Swagger 규칙에 맞게 마무리합니다.
>
> <details><summary>Created a file, deleted a file</summary>
>
> - Deleted `./internal/handler/employeeHandler.go` (+0 -22)
> - Wrote `./internal/handler/employeeHandler.go` (+249 -0)
>
> </details>
>
> Handler 라우팅이 완성됐으니, `main`에서 등록과 서버 시작까지 연결하겠습니다.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./cmd/gocore/main.go` (+2 -2)
>
> </details>
>
> 이제 컴파일 관점에서 한 번 다듬습니다. 특히 GORM 테이블명과 import 정리가 꼬일 수 있는 부분을 확인할게요.
>
> 환경에 `gofmt`가 없네요. Go 툴체인 위치를 확인하고, 없으면 정적 확인 위주로 진행하겠습니다.
>
> 한 가지 DB 테이블명 매핑을 놓치면 런타임에서 얄밉게 터집니다. 스키마와 GORM 이름을 정확히 맞춰두겠습니다.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./internal/domain/employee.go` (+16 -0)
>
> </details>
>
> Go 툴체인이 PATH에 없어서 직접 빌드는 못 했습니다. 대신 문법상 눈에 띄는 불필요 코드와 포맷을 손으로 정리하겠습니다.
>
> 툴체인은 `/usr/local/go/bin`에 있네요. 불필요 helper 제거 후 실제 포맷/테스트를 돌리겠습니다.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./internal/service/employeeService.go` (+0 -17)
>
> </details>
>
> 이제 진짜 Go 포맷과 테스트입니다. DB 접속 없이 컴파일 단계까지 확인될 거예요.
>
> 컴파일은 통과했는데 Go 캐시 정리만 홈 디렉터리 read-only에 걸렸습니다. 캐시를 `/tmp`로 돌려 재확인할게요.
>
> 테스트 통과했습니다. 마지막으로 변경 요약을 정확히 잡기 위해 diff 통계를 확인하겠습니다.
>
> 흥미롭게도 새 코드 파일들이 아직 Git 추적 전 상태라 diff 통계에 안 잡히네요. 상태를 정확히 확인하겠습니다.
>
> 검토 중 두 가지 행동 차이를 바로잡겠습니다: 직원 로그인에서 역할 불일치는 401로, 계정 생성 시 비밀번호 변경시각은 현재시각으로 저장되게요.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./internal/service/employeeService.go` (+8 -4)
>
> </details>
>
> 작은 동작 보정까지 끝났습니다. 포맷과 컴파일을 다시 한 번 돌려 마침표 찍겠습니다.
>
> 통과했습니다. 최종 요약에 정확한 파일 위치를 담기 위해 라인 번호만 확인하겠습니다.
>
> 정리와 검증까지 완료했습니다. 계획도 마무리 표시하겠습니다.
</details>

완성했습니다. Swagger 기준으로 Handler → Service → Repository 레이어를 연결했고 컴파일 확인도 통과했어요.

**변경 사항**
- `internal/domain/employee.go:19` GORM 모델, 요청/응답 DTO, 레이어 인터페이스 정의
- `internal/repository/employeeRepository.go:20` 로그인 계정 조회, 직원 CRUD, 퇴사/블랙리스트 DB 처리 구현
- `internal/service/employeeService.go:66` 직원/관리자 로그인, 토큰 인증, 권한 체크, 인메모리 블랙리스트 구현
- `internal/service/employeeService.go:202` 직원 상세 조회 시 Background Check API 재시도/백오프 로직 구현
- `internal/handler/employeeHandler.go:41` `emp_swagger.yaml`의 전체 Echo 라우트 등록
- `internal/handler/employeeHandler.go:211` 실패 응답 규칙 반영: `BLACKLISTED_EMPLOYEE`만 `code` 포함

**연결**
- `cmd/gocore/main.go:18` Echo 서버에 라우트 등록 후 `:8080` 실행

**검증**
- 실행: `GOCACHE=/tmp/go-build-cache /usr/local/go/bin/go test ./...`
- 결과: 전체 패키지 컴파일 통과

참고로 `BACKGROUND_CHECK_API_URL`과 `AUTH_SECRET` 환경변수를 쓰도록 해뒀습니다.

> UserAccount와 같이 gorm 모델 구조체의 tag에 column이 빠져있어. [schema.sql](schema.sql)를 참고하여 전부 채워줘

<details><summary>4 previous messages</summary>

> 좋아요, `schema.sql` 컬럼명과 GORM 태그를 1:1로 맞춰두겠습니다. 이런 건 초반에 잡아두면 뒤에서 덜 삐걱거려요.
>
> 컬럼 매핑 대상은 네 개 모델입니다. 이제 tag에 `column:*`을 빠짐없이 채우고 포맷/테스트까지 돌리겠습니다.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./internal/domain/employee.go` (+34 -34)
>
> </details>
>
> 태그를 채웠습니다. 이제 포맷하고 컴파일 확인해서 매핑 변경이 다른 곳을 깨지 않았는지 보겠습니다.
</details>

완료했습니다.

- `internal/domain/employee.go:19` `Department` 모든 필드에 `column` 태그 추가
- `internal/domain/employee.go:31` `Employee` 모든 DB 컬럼 필드에 `column` 태그 추가
- `internal/domain/employee.go:53` `UserAccount` 모든 DB 컬럼 필드에 `column` 태그 추가
- `internal/domain/employee.go:72` `TerminatedEmployeeBlacklist` 모든 DB 컬럼 필드에 `column` 태그 추가

검증도 통과했습니다.

- `GOCACHE=/tmp/go-build-cache /usr/local/go/bin/go test ./...`

> os.Getenv 실패 시 Fatal에러를 내줘

<details><summary>4 previous messages</summary>

> 좋아요, 환경변수 누락을 조용히 넘기지 않고 시작 단계에서 바로 죽도록 바꾸겠습니다. 설정 오류는 빨리 터지는 게 착합니다.
>
> `AUTH_SECRET`와 `BACKGROUND_CHECK_API_URL` 두 곳이 대상입니다. 둘 다 필수 환경변수로 바꾸고 fallback 로직을 제거할게요.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./internal/service/employeeService.go` (+11 -18)
>
> </details>
>
> 필수 환경변수 처리로 바꿨습니다. 포맷과 컴파일만 빠르게 확인하겠습니다.
</details>

완료했습니다.

- `internal/service/employeeService.go:52` `BACKGROUND_CHECK_API_URL` 누락 시 `log.Fatalf`
- `internal/service/employeeService.go:55` `AUTH_SECRET` 누락 시 `log.Fatalf`
- `internal/service/employeeService.go:492` `requiredEnv` helper 추가
- 기존 `AUTH_SECRET` 기본값과 `BACKGROUND_CHECK_API_URL` 미설정 fallback 제거

검증도 통과했습니다.

- `GOCACHE=/tmp/go-build-cache /usr/local/go/bin/go test ./...`

> jwt 토큰생성/검증은 golang-jwt latest 버전 사용해

<details><summary>16 previous messages</summary>

> 좋아요, 직접 만든 HMAC 토큰을 걷어내고 `golang-jwt/jwt` 최신 v5 계열로 바꾸겠습니다. 먼저 의존성 추가를 시도하고 막히면 승인 요청으로 이어갈게요.
>
> <details><summary>Ran 2 commands</summary>
>
> - Ran `/usr/local/go/bin/go get github.com/golang-jwt/jwt/v5@latest`
> - Ran `/usr/local/go/bin/go get github.com/golang-jwt/jwt/v5@latest`
>
> </details>
>
> `jwt/v5` 최신 의존성이 들어왔습니다. 이제 Service의 서명/검증 구현을 라이브러리 기반으로 교체하겠습니다.
>
> <details><summary>Explored 3 files</summary>
>
> - Read `./internal/service/employeeService.go`
> - Read `./internal/service/employeeService.go`
> - Read `./go.mod`
>
> </details>
>
> 현재 수제 토큰 함수가 남아있습니다. imports와 claims 타입까지 `jwt/v5` 방식으로 한 번에 갈아끼울게요.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./internal/service/employeeService.go` (+18 -32)
>
> </details>
>
> `ExpiresAt` 타입이 JWT 표준 클레임으로 바뀌었으니 검증부도 맞춰야 합니다. 남은 타입 차이를 정리하겠습니다.
>
> <details><summary>Explored a search</summary>
>
> - Searched for `ExpiresAt|tokenSignature|base64|hmac|sha256|github.com/golang-jwt` in `employeeService.go`
>
> </details>
>
> 찾은 남은 수제 만료 검증 한 줄을 JWT 클레임 기준으로 바꿉니다. 의존성도 직접 사용이니 `indirect`를 제거하겠습니다.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./internal/service/employeeService.go` (+1 -1)
>
> </details>
>
> `go.mod`도 직접 의존성으로 정리하고 테스트하겠습니다.
>
> <details><summary>Edited a file</summary>
>
> - Wrote `./go.mod` (+1 -1)
>
> </details>
>
> 이제 포맷과 전체 테스트를 돌려 JWT 교체가 깨끗한지 확인합니다.
>
> <details><summary>Ran a command</summary>
>
> - Ran `/usr/local/go/bin/gofmt -w internal/service/employeeService.go && GOCACHE=/tmp/go-build-cache /usr/local/go/bin/go test ./...`
>
> </details>
>
> 테스트 통과했습니다. 마지막으로 변경 위치를 잡아두겠습니다.
>
> <details><summary>Ran a command</summary>
>
> - Ran `nl -ba internal/service/employeeService.go | sed -n '1,115p;270,345p'; nl -ba go.mod | sed -n '1,30p'; git status --short`
>
> </details>
</details>

완료했습니다.

- `go.mod:6` `github.com/golang-jwt/jwt/v5 v5.3.1` 추가
- `internal/service/employeeService.go:37` 커스텀 claims가 `jwt.RegisteredClaims`를 사용하도록 변경
- `internal/service/employeeService.go:278` 토큰 생성 로직을 `jwt.NewWithClaims(jwt.SigningMethodHS256, claims)`로 교체
- `internal/service/employeeService.go:68` 토큰 검증 로직을 `jwt.ParseWithClaims` 기반으로 교체
- 기존 수제 HMAC/base64 토큰 생성·검증 코드 제거

검증도 통과했습니다.

- `GOCACHE=/tmp/go-build-cache /usr/local/go/bin/go test ./...`
