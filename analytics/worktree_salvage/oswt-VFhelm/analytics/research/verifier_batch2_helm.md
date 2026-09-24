# verifier_batch2_helm — hidden black-box suites

Verifier-author pass. Suites live in `src/openswe_traces/synth/testdata/batch2_helm/` and are packaged to `experiments/pipeline/tasks_batch2/helm/<unit>-L{0,2}/`.
Worked from `api.md` + `DETAILS.md` + excised trees only. Gold/contract/bugreport unread.

Overlap check (`scripts/check_unit_overlap.py --extra /home/evan/Documents/oswt-AUhelm` from the main checkout): **CLEAN**, 14 new vs 10 existing, 0 overlaps. No units excluded.

Hidden seed: `HIDDEN_SEED` env else `20260919`. Gold proved under both 20260919 and 20260920. B4 packaging check passed on every L0/L2 dir. Cursor host allowlist in `task.toml`. Verifier `network_mode = no-network`.

In-image preflight: `oswt-closureO` `scripts/preflight_task.py` (bare must FAIL with assertions/panic, gold PASS, cheat FAIL). 28/28 passed.

## Preflight table

| unit | details | tests | L0 bare/gold/cheat | L0 s | L2 bare/gold/cheat | L2 s | verdict |
|---|---:|---:|---|---:|---|---:|---|
| chartmeta | 9 | 9 | fail/pass/fail | 59.1 | fail/pass/fail | 59.1 | pass |
| searchindex | 10 | 10 | fail/pass/fail | 60.7 | fail/pass/fail | 61.1 | pass |
| chartdl | 11 | 11 | fail/pass/fail | 78.4 | fail/pass/fail | 78.9 | pass |
| dlmanager | 11 | 11 | fail/pass/fail | 78.7 | fail/pass/fail | 78.7 | pass |
| getterdispatch | 8 | 8 | fail/pass/fail | 60.7 | fail/pass/fail | 60.8 | pass |
| httpgetter | 8 | 8 | fail/pass/fail | 60.2 | fail/pass/fail | 60.5 | pass |
| urlutil | 6 | 6 | fail/pass/fail | 23.0 | fail/pass/fail | 0.0 | pass |
| sympath | 7 | 7 | fail/pass/fail | 29.0 | fail/pass/fail | 29.0 | pass |
| tlsutil | 7 | 7 | fail/pass/fail | 27.8 | fail/pass/fail | 28.1 | pass |
| copyst | 8 | 8 | fail/pass/fail | 29.7 | fail/pass/fail | 30.2 | pass |
| valuesopts | 8 | 8 | fail/pass/fail | 50.1 | fail/pass/fail | 55.2 | pass |
| helmpath | 5 | 5 | fail/pass/fail | 24.5 | fail/pass/fail | 24.7 | pass |
| chartrepo | 9 | 9 | fail/pass/fail | 81.7 | fail/pass/fail | 82.4 | pass |
| relsplit | 8 | 8 | fail/pass/fail | 48.0 | fail/pass/fail | 48.4 | pass |

Total details/tests: 115. Packaged: 28 dirs. Excluded: none.

## Per-unit tests

### chartmeta (`pkg/chart/v2`, 9 details)

Hidden file: `tests/hidden/pkg/chart/v2/chartmeta_bb_prop_test.go`

- `TestDetail01_SanitizeInPlaceUnicodeSpaceAndNonPrintable` ← DETAILS.md line 1
- `TestDetail02_CheckOrderAPIVersionNameVersionType` ← DETAILS.md line 2
- `TestDetail03_ReservedNamesAndBasename` ← DETAILS.md line 3
- `TestDetail04_LenientSemver` ← DETAILS.md line 4
- `TestDetail05_TypeEnumEmptyApplicationLibrary` ← DETAILS.md line 5
- `TestDetail06_AliasCharsetAndErrorNamesDependency` ← DETAILS.md line 6
- `TestDetail07_DuplicateKeyedOnAliasElseName` ← DETAILS.md line 7
- `TestDetail08_NilReceiversNoPanic` ← DETAILS.md line 8
- `TestDetail09_ValidationErrorPrefixAndFormat` ← DETAILS.md line 9

### searchindex (`pkg/cmd/search`, 10 details)

Hidden file: `tests/hidden/pkg/cmd/search/searchindex_bb_prop_test.go`

- `TestDetail01_MatchLineFourFieldsVerticalTab` ← DETAILS.md line 1
- `TestDetail02_ScoreIsFieldIndexSeparatorBelongsNext` ← DETAILS.md line 2
- `TestDetail03_LiteralLowercasesRegexpCaseSensitive` ← DETAILS.md line 3
- `TestDetail04_ThresholdExclusive` ← DETAILS.md line 4
- `TestDetail05_BadRegexpEmptyNonNilSliceAndError` ← DETAILS.md line 5
- `TestDetail06_AddRepoJoinSkipEmptySortEntriesMutates` ← DETAILS.md line 6
- `TestDetail07_AllFalseNewestOnlyAllTrueEveryVersion` ← DETAILS.md line 7
- `TestDetail08_DollarDollarStrippedFromNames` ← DETAILS.md line 8
- `TestDetail09_SortScoreScoreThenNameThenSemverDesc` ← DETAILS.md line 9
- `TestDetail10_AllScoreZeroOnePerKey` ← DETAILS.md line 10

### chartdl (`pkg/downloader`, 11 details)

Hidden file: `tests/hidden/pkg/downloader/chartdl_bb_prop_test.go`

- `TestDetail01_OCIShortCircuitTagDigestRules` ← DETAILS.md line 1
- `TestDetail02_AbsoluteURLScanSwallowsErrNoOwnerRepo` ← DETAILS.md line 2
- `TestDetail03_RepoChartSplitAndErrors` ← DETAILS.md line 3
- `TestDetail04_IndexLookupEmptyVersionLatestURLResolve` ← DETAILS.md line 4
- `TestDetail05_DigestAlgoPrefixAnd32ByteCheck` ← DETAILS.md line 5
- `TestDetail06_DestFilenameBasenameOCIColonToDash` ← DETAILS.md line 6
- `TestDetail07_ProvVerifyStrategies` ← DETAILS.md line 7
- `TestDetail08_DownloadToCacheHitMissAndDigestKey` ← DETAILS.md line 8
- `TestDetail09_VerifyChartPreconditions` ← DETAILS.md line 9
- `TestDetail10_IsTarCaseInsensitiveTgzOnly` ← DETAILS.md line 10
- `TestDetail11_GetterOptionsAccumulate` ← DETAILS.md line 11

### dlmanager (`pkg/downloader`, 11 details)

Hidden file: `tests/hidden/pkg/downloader/dlmanager_bb_prop_test.go`

- `TestDetail01_ResolveRepoNamesDispatchOrder` ← DETAILS.md line 1
- `TestDetail02_MissingRepoErrorPleaseAddAndNote` ← DETAILS.md line 2
- `TestDetail03_HasAllReposTrailingSlashAndErrRepoNotFound` ← DETAILS.md line 3
- `TestDetail04_ErrRepoNotFoundErrorNoPleaseAdd` ← DETAILS.md line 4
- `TestDetail05_EnsureMissingReposHelmManagerSHA256` ← DETAILS.md line 5
- `TestDetail06_FindChartURLConfiguredAndOCIAndFallback` ← DETAILS.md line 6
- `TestDetail07_FindVersionedEntryEmptyVersionFirstWithURLs` ← DETAILS.md line 7
- `TestDetail08_VersionEqualsAsymmetry` ← DETAILS.md line 8
- `TestDetail09_ParseOCIRefPortNotTag` ← DETAILS.md line 9
- `TestDetail10_KeyHexSHA256` ← DETAILS.md line 10
- `TestDetail11_DedupeReposTrailingSlashLastWins` ← DETAILS.md line 11

### getterdispatch (`pkg/getter`, 8 details)

Hidden file: `tests/hidden/pkg/getter/getterdispatch_bb_prop_test.go`

- `TestDetail01_ProvidesExactSchemeMembership` ← DETAILS.md line 1
- `TestDetail02_BySchemeFirstMatchAndErrorText` ← DETAILS.md line 2
- `TestDetail03_BuiltinsHTTPHTTPSOneProviderPlusOCI` ← DETAILS.md line 3
- `TestDetail04_ConstructorOptionOrderCallThenDefaultThenExtras` ← DETAILS.md line 4
- `TestDetail05_AllSwallowsDiscoveryErrorAddsPlugins` ← DETAILS.md line 5
- `TestDetail06_OnlyGetterV1PluginsContribute` ← DETAILS.md line 6
- `TestDetail07_ConvertOptionsCallBeatsGlobalSubset` ← DETAILS.md line 7
- `TestDetail08_PluginGetSchemeProtocolAndErrorWrap` ← DETAILS.md line 8

### httpgetter (`pkg/getter`, 8 details)

Hidden file: `tests/hidden/pkg/getter/httpgetter_bb_prop_test.go`

- `TestDetail01_GetCopiesThenOverlaysNoMutation` ← DETAILS.md line 1
- `TestDetail02_AcceptIfSetUADefaultAndOverride` ← DETAILS.md line 2
- `TestDetail03_BasicAuthScopePassAllOrSameHostPort` ← DETAILS.md line 3
- `TestDetail04_DualParseErrorWording` ← DETAILS.md line 4
- `TestDetail05_Non200ErrorSpacesAroundColon` ← DETAILS.md line 5
- `TestDetail06_NeedsCustomTLSFormula` ← DETAILS.md line 6
- `TestDetail07_TransportPrecedenceExplicitThenCustomThenShared` ← DETAILS.md line 7
- `TestDetail08_DisableCompressionTimeoutAlwaysApplied` ← DETAILS.md line 8

### urlutil (`internal/urlutil`, 6 details)

Hidden file: `tests/hidden/internal/urlutil/urlutil_bb_prop_test.go`

- `TestDetail01_URLJoinPathOnlyPreservesSchemeHostQuery` ← DETAILS.md line 1
- `TestDetail02_URLJoinPathishBaseAndUnparseableError` ← DETAILS.md line 2
- `TestDetail03_EqualNormalizesEmptyPathAndCleansDots` ← DETAILS.md line 3
- `TestDetail04_EqualFallbackAsymmetry` ← DETAILS.md line 4
- `TestDetail05_ExtractHostnameStripsPort` ← DETAILS.md line 5
- `TestDetail06_URLJoinZeroComponentsUnchanged` ← DETAILS.md line 6

### sympath (`internal/sympath`, 7 details)

Hidden file: `tests/hidden/internal/sympath/sympath_bb_prop_test.go`

- `TestDetail01_WalkLexicalOrderIncludesRoot` ← DETAILS.md line 1
- `TestDetail02_SymlinkVisitAtLinkPathResolvedInfo` ← DETAILS.md line 2
- `TestDetail03_SkipDirSwallowedOnResolvedLink` ← DETAILS.md line 3
- `TestDetail04_SkipDirOnPlainFilePropagates` ← DETAILS.md line 4
- `TestDetail05_RootStatErrorCallbackSkipDirNil` ← DETAILS.md line 5
- `TestDetail06_ReadDirFailureGoesToCallback` ← DETAILS.md line 6
- `TestDetail07_IsSymlinkModeBit` ← DETAILS.md line 7

### tlsutil (`internal/tlsutil`, 7 details)

Hidden file: `tests/hidden/internal/tlsutil/tlsutil_bb_prop_test.go`

- `TestDetail01_OptionsRunAllErrorsJoined` ← DETAILS.md line 1
- `TestDetail02_CertKeyPairEmptyNoopVsSinglePath` ← DETAILS.md line 2
- `TestDetail03_CAFileEmptyNoopVsReadError` ← DETAILS.md line 3
- `TestDetail04_CertificateOnlyWhenBothPEMBlocks` ← DETAILS.md line 4
- `TestDetail05_CAAppendFailure` ← DETAILS.md line 5
- `TestDetail06_InsecurePassthroughNilRootCAs` ← DETAILS.md line 6
- `TestDetail07_NoOptionsValidConfig` ← DETAILS.md line 7

### copyst (`internal/copystructure`, 8 details)

Hidden file: `tests/hidden/internal/copystructure/copyst_bb_prop_test.go`

- `TestDetail01_CopyNilYieldsEmptyMap` ← DETAILS.md line 1
- `TestDetail02_ScalarsAndArraysReturnedAsIs` ← DETAILS.md line 2
- `TestDetail03_InterfaceNilPreservedDynamicCopied` ← DETAILS.md line 3
- `TestDetail04_MapNilAndNilInterfaceValues` ← DETAILS.md line 4
- `TestDetail05_PointerNilOrRewrap` ← DETAILS.md line 5
- `TestDetail06_SliceLenAndCapNilElems` ← DETAILS.md line 6
- `TestDetail07_StructFieldWiseUnexportedPanics` ← DETAILS.md line 7
- `TestDetail08_FuncChanSharedUnsupportedKindErrors` ← DETAILS.md line 8

### valuesopts (`pkg/cli/values`, 8 details)

Hidden file: `tests/hidden/pkg/cli/values/valuesopts_bb_prop_test.go`

- `TestDetail01_LayerOrderFilesJSONSetStringFileLiteral` ← DETAILS.md line 1
- `TestDetail02_SetJSONBraceSniffVsParseJSON` ← DETAILS.md line 2
- `TestDetail03_SetJSONErrorTextAsymmetry` ← DETAILS.md line 3
- `TestDetail04_DashAfterTrimSpaceIsStdin` ← DETAILS.md line 4
- `TestDetail05_UnsupportedSchemeFallsBackToLocalFile` ← DETAILS.md line 5
- `TestDetail06_RemoteReadPassesWithURL` ← DETAILS.md line 6
- `TestDetail07_ValueFilesReadLoadMergeInFlagOrder` ← DETAILS.md line 7
- `TestDetail08_PerFamilyErrorWrap` ← DETAILS.md line 8

### helmpath (`pkg/helmpath`, 5 details)

Hidden file: `tests/hidden/pkg/helmpath/helmpath_bb_prop_test.go`

- `TestDetail01_HelmEnvOmitsHelmSubdir` ← DETAILS.md line 1
- `TestDetail02_XDGOrDefaultInsertsHelmSubdir` ← DETAILS.md line 2
- `TestDetail03_PrecedenceHelmOverXDGOverDefaultEmptyUnset` ← DETAILS.md line 3
- `TestDetail04_LazyEnvEvalPerCall` ← DETAILS.md line 4
- `TestDetail05_CacheIndexAndChartsFileNameDash` ← DETAILS.md line 5

### chartrepo (`pkg/repo/v1`, 9 details)

Hidden file: `tests/hidden/pkg/repo/v1/chartrepo_bb_prop_test.go`

- `TestDetail01_NewChartRepositoryURLAndSchemeErrors` ← DETAILS.md line 1
- `TestDetail02_DownloadIndexResolvesIndexYAMLPassesTLSAuth` ← DETAILS.md line 2
- `TestDetail03_WritesChartsTxtAndIndexYAML` ← DETAILS.md line 3
- `TestDetail04_FindChartInRepoURLRandomNameCleanup` ← DETAILS.md line 4
- `TestDetail05_ErrorShapesNotRepoVersionClauseNoURLs` ← DETAILS.md line 5
- `TestDetail06_ResolveReferenceURLAbsolutePassthrough` ← DETAILS.md line 6
- `TestDetail07_PathAndRawPathSlashNormalize` ← DETAILS.md line 7
- `TestDetail08_BaseRawQueryCarriedOntoResolved` ← DETAILS.md line 8
- `TestDetail09_IndexLoadViaDownload` ← DETAILS.md line 9

### relsplit (`pkg/release/v1/util`, 8 details)

Hidden file: `tests/hidden/pkg/release/v1/util/relsplit_bb_prop_test.go`

- `TestDetail01_CheckNilReleaseFalseWithoutInvoking` ← DETAILS.md line 1
- `TestDetail02_StatusFilterInnerNilTrue` ← DETAILS.md line 2
- `TestDetail03_AnyOrEmptyFalseAllAndEmptyTrue` ← DETAILS.md line 3
- `TestDetail04_FilterPreservesOrder` ← DETAILS.md line 4
- `TestDetail05_SplitRegexLineStartFusedText` ← DETAILS.md line 5
- `TestDetail06_LeadingWhitespaceTrimmedBeforeSplit` ← DETAILS.md line 6
- `TestDetail07_EmptyDocsDroppedKeysManifestN` ← DETAILS.md line 7
- `TestDetail08_BySplitManifestsOrderNumericSuffix` ← DETAILS.md line 8

## Packaging notes

- First preflight wave failed gold/cheat with `PATCH_APPLY_FAILED` because `package_unit` reused a scratch tree that iterate had already patched. Fixed by rematerializing the excised tree (`force=True`) before packaging. Re-packaged; 28/28 preflight pass.
- Bare failures are excised panics (`panic: excised: …`) or assertion FAILs, never setup/build/import errors.
- Cheat fails the property suite (A3).

Log: `outputs/VFhelm.log`.
