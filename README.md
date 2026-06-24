# employee-manage


https://app.swaggerhub.com/apis/wlsrud0303organizati/background-check-api/1.0.0#/default/getBackgroundCheck

https://app.swaggerhub.com/apis/wlsrud0303organizati/employee-management-rest-api/1.0.0#/Employee%20Auth/loginEmployee




- Language: `Go`
- DB: `PostgresDB` (`GORM`)
- Code Architecture: `Clean Architecture`




```
[주요 기능]
(참고: UI는 직원용/관리자용으로 나눠져 있음)
1. 직원 로그인
2. 직원이 자신의 인적사항을 조회
3. 직원이 자신의 인적사항을 수정
4. 관리자 로그인
5. 관리자가 직원 계정 생성/퇴사 처리
* 퇴사처리되면 즉시 차단
6. 관리자가 전체 직원 목록 조회, 개별 상세정보 조회(상세정보에는 직원의 배경 정보도 포함됨)
* 직원의 배경정보는 Background Check API로 조회


[주의할 점]
1. 클라이언트에게 보내는 성공/실패 응답 객체는 통일. (예. timestamp, message, code)
2. Background Check API 요청 실패 핸들링
-  최대 3번까지 재요청하고 실패시 500 에러. 재시도 시 지수백오프+jitter 섞어서 재시도.
 - status 503이고 response에 retryAfter가 있는 경우: 해당 retryAfter 값만큼 기다렸다가 Retry 최대 3번.
3. 직원 블랙리스트를 서버 인메모리 캐싱. 요청 시 마다 확인하고 등록된 유저면 401 리턴.
```