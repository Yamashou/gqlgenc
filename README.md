# gqlgenc

## What is gqlgenc ?

This is Go library for building GraphQL client with [gqlgen](https://github.com/99designs/gqlgen)

## Motivation

Now, if you build GraphQL api client for Go, have choice:
 
 - [github.com/shurcooL/graphql](https://github.com/shurcooL/graphql)
 - [github.com/machinebox/graphql](https://github.com/machinebox/graphql)

These libraries are very simple and easy to handle. 
However, as I work with [gqlgen](https://github.com/99designs/gqlgen) and [graphql-code-generator](https://graphql-code-generator.com/) every day, I find out the beauty of automatic generation.
So I want to automatically generate types. 

## Installation

```shell script
go get -u github.com/Yamashou/gqlgenc
```

## How to use

gqlgenc base is gqlgen with [plugins](https://gqlgen.com/reference/plugins/). So the setting is yaml in each format.
gqlgenc can be configured using a .gqlgenc.yml file, 

```yaml

model:
  package: generated
  filename: ./models_gen.go # https://github.com/99designs/gqlgen/tree/master/plugin/modelgen
client:
  package: generated
  filename: ./client.go # Where should any generated client go?
models:
  Int:
    model: github.com/99designs/gqlgen/graphql.Int64
  Date:
    model: github.com/99designs/gqlgen/graphql.Time
endpoint:
  url: https://api.annict.com/graphql # Where do you want to send your request?
  headers:　# If you need header for getting introspection query, set it 
    Authorization: "Bearer xxxxxxxxxxxxx"
query:
  - "./query/*.graphql" # Where are all the query files located? 
```


```shell script
gqlgenc
```



#### documents

- [How to configure gqlgen using gqlgen.yml](https://gqlgen.com/config/)



