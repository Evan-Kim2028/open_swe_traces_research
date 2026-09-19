from openswe_traces.synth.repos2 import _parse_test_packages


def test_parse_go_test_json_package_events() -> None:
    text = """
{"Action":"run","Package":"example.internal/clikit"}
{"Action":"pass","Package":"example.internal/clikit","Test":"TestA","Elapsed":0.01}
{"Action":"pass","Package":"example.internal/clikit","Elapsed":0.12}
{"Action":"fail","Package":"example.internal/httprouter","Elapsed":1.5}
{"Action":"skip","Package":"example.internal/msgbus/logger","Elapsed":0}
ok  \texample.internal/clikit\t0.123s
"""
    rows = {r["import_path"]: r for r in _parse_test_packages(text)}
    assert rows["example.internal/clikit"]["status"] == "ok"
    assert rows["example.internal/httprouter"]["status"] == "fail"
    assert rows["example.internal/msgbus/logger"]["status"] == "skip"
    assert rows["example.internal/clikit"]["elapsed_sec"] == 0.12
