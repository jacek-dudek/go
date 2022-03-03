# [OpenM++](http://www.openmpp.org/) Go tools

This repository is a part of [OpenM++](http://www.openmpp.org/) open source microsimulation platform.
It contains oms web-service, dbcopy utility and openM++ Go libraries.

## Build

Build from GitHub master (use this for initial build):
```
go install github.com/openmpp/go/dbcopy@latest
go install github.com/openmpp/go/oms@latest
```

Build from your local sources (use this if made some changes):
```
git clone https://github.com/openmpp/go ompp-go
cd ompp-go
go install ./dbcopy
go install ./oms
```

By default only SQLite database supported. 
If you want to use other database vendors (Microsoft SQL, MySQL, PostgreSQL, IBM DB2, Oracle) then compile dbcopy with ODBC support:

```
go install -tags odbc github.com/openmpp/go/dbcopy@latest
```

Please visit our [wiki](https://github.com/openmpp/openmpp.github.io/wiki) for more information or e-mail to: _openmpp dot org at gmail dot com_.

**License:** MIT.
