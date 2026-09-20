# Closure — rootval

`expr/root.go` — root expression: eval walk order, design-level validation
(duplicates, shared error names, shared routes, relocated types, type
mappings), finalize, meta-map helpers.

Symbols stubbed: `(*RootExpr).WalkSets`, `(*RootExpr).UserType`,
`(*RootExpr).Service`, `(*RootExpr).Error`, `(*RootExpr).Validate`,
`ValidateSharedErrorNames`, `hasErrorNameAttribute`,
`(*RootExpr).validateErrorDefaults`, `(*RootExpr).validateSharedHTTPRoutes`,
`routesMountedOnServer`, `serverHostsService`,
`routePatternWithoutParameterNames`, `validateTypeMappings`,
`(*RootExpr).validateRelocatedUserTypes`,
`(*RootExpr).walkUserTypeDependencies`,
`(*RootExpr).walkAttributeUserTypes`, `(*RootExpr).Finalize`,
`(*RootExpr).walkHTTPServices`, `MetaExpr.Dup`, `MetaExpr.Merge`,
`MetaExpr.Last`. (`Packages`, `DependsOn`, `EvalName` are left implemented —
they run at package init.)

Tests removed: `expr/root_test.go` deleted (pins relocated-type diagnostics,
mixed-route detection, type-mapping dedupe, MetaExpr.Last).
