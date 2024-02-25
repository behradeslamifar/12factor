# 12factor
This code was written with the help of ChatGPT.
Read [12factor.net](https://12factor.net/) first.

## How to test the project
Start database
```
docker-compose up -d
```

Create database and tables
```
mysql> source database.sql;
```

Run project
```
go run main.go
```
