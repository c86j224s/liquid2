# Goal Preservation Audit

Broad-open narrowing harm is scored separately from the process composite:

- 0: raw instruction breadth preserved and needed perspective expansion performed.
- 1: some perspective missing but core breadth preserved.
- 2: `run_goal` or step instruction clearly reduced the possibility space.
- 3: the run effectively followed a narrow goal that conflicts with the raw instruction.

Hard-failure goal-preservation cases:

- `user_instruction_raw` omitted from `S1-layered` prompt or manifest.
- `run_goal` described as replacing or overriding the raw instruction.
- `step_instruction` injects a conclusion, fact, citation, or report paragraph.
- Generated result, controller output, report, run goal, or step instruction is treated as source.
