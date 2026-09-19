# Closure — connarray

Package: internal/client.

Removed functions (bodies stubbed): newConnArray, connArray.Init, connArray.Get, connArray.Close, monitoredDial, monitoredConn.Close, connMonitor.start.

Exported entry point(s): RPCClient.SendRequest -> getConnArray -> connArray.Init/Get.
