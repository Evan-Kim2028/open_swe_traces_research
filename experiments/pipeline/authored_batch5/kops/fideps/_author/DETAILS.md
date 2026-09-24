# Details — fideps

1. `NotADependency.GetDependencies` returns nil — embedding it opts a struct out of the
   reflective scan without implementing task semantics. Inferable: yes.
2. `FindTaskDependencies` prefers the `HasDependencies` interface over reflection —
   declared deps win. Inferable: yes.
3. Dependencies are reported as task KEYS (map keys), not tasks — the `taskToId` lookup is
   by task identity (interface equality). Inferable: yes.
4. Nil dependencies (typed or untyped nil) are silently skipped; a non-nil dep missing from
   the map is `klog.Fatalf`, not an error return. Inferable: partially — fatal vs error.
5. The reflective walk ignores the task's own root struct (`path.IsEmpty()`), primitives,
   strings, and descends through ptr/interface/slice/map without treating containers as deps.
   Inferable: yes.
6. A struct field implementing BOTH `HasDependencies` and `Task` contributes its declared
   deps AND itself. A struct implementing only `Resource` is ignored. Any other non-task
   struct type is an error (`Unhandled type`). Inferable: partially — these precedence rules
   are specific.
7. `FindDependencies` applies the same logic to non-task objects. Inferable: yes.
