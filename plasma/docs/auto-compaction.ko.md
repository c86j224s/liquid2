# 자동 압축 범위

Plasma의 자동 압축은 provider가 모델 컨텍스트가 소진되었다고 보고한 **재개된 Web 대화 턴**에만 적용됩니다. 기존 provider session을 유지하면서 압축 요청을 한 번 실행하고, 압축된 session 이벤트를 append한 뒤 원래 요청을 한 번 재시도합니다.

공유 conversation capability는 자동 압축 조건, 엄격한 동일 session 검증, 단계 결과, 경과 시간, callback 순서를 소유합니다. Provider 실행, live observation, ledger 저장은 소유하지 않습니다. Workflow 실행의 별도 빈 session fallback과 기존 선제적 압축 동작은 변경하지 않습니다.

자동 압축은 새 session을 만들지 않습니다. 반환 session ID가 비어 있거나 다르면 invalid input으로 기록하며, 압축 이벤트 append가 실패하면 재시도하지 않습니다. 수동 압축은 별도의 Web 경로로 유지됩니다.
