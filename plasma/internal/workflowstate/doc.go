// Package workflowstate는 workflow 장부 이벤트의 stable payload와 projection
// 규칙을 정의한다.
//
// 이 패키지는 이벤트 열을 WorkflowRunView로 투영하고 terminal 상태를 판정한다.
// 이벤트를 언제 append할지, 어떤 agent를 실행할지는 workflowruns service와 runner
// 계층이 결정한다.
package workflowstate
