# client

## Remove MarshalJSON function
clientはclient.MarshalJSONに相当するものはありません。代わりに json.Marshal を使用します。
そのためenumの型には必ずjson.Marshalerの実装が必要になります。
gqlgen v1.17.71以降では、enumの型にjson.Marshalerのコードが生成されます。
```go
func (e *EnumValue) UnmarshalJSON(b []byte) error {
       s, err := strconv.Unquote(string(b))
       if err != nil {
               return err
       }
       return e.UnmarshalGQL(s)
}

func (e EnumValue) MarshalJSON() ([]byte, error) {
       var buf bytes.Buffer
       e.MarshalGQL(&buf)
       return buf.Bytes(), nil
}
```

- Reference
  - https://github.com/99designs/gqlgen/pull/3663

## Support omitzero
Go 1.24に導入されたomitzeroに対応しました。
Go公式ではomitemptyではなくomitzeroが推奨されています。
https://pkg.go.dev/github.com/go-json-experiment/json#example-package-OmitFields
gqlgenでomitzeroを設定するにはi `gqlgen.yaml` 以下の設定をしてください。
```yaml
enable_model_json_omitzero_tag: true
# omitemptyを無効にしたい場合はfalseにしてください。
enable_model_json_omitempty_tag: false
```

- Reference
 - https://github.com/99designs/gqlgen/pull/3659

## nullとundefinedの区別がつけられるようになりました
APIによっては、nullとundefinedで振る舞いが異なる場合があります。
以下のようにnullとundefinedを区別できるようになりました。

- json: `{}` means undefined, no effect to name
```go
input := UserUpdateInput{
		Name:         graphql.Omittable[string]{},
}
```

- json: `{"name":null}` meanas null, set empty string to  name
```go
input := UserUpdateInput{
		Name:         graphql.OmittableOf[string](null),
}
```

Omittable型は `gqlgen.yaml` `nullable_input_omittable: ture` にすることで生成されます。またomitするためには `enable_model_json_omitzero_tag: true` にしてomitzeroタグを付与してください。
undefined時にomitするためには、gqlgen v1.17.71以降を利用してください。Omittable型にIsZeroメソッドが追加されました。

```yaml
nullable_input_omittable: true

enable_model_json_omitzero_tag: true
enable_model_json_omitempty_tag: false
```

```go
type UserUpdateInput struct {
    Name graphql.Omittable[*string] `json:"name,omitzero"`
```

- Reference
  - https://github.com/99designs/gqlgen/pull/3660


## Support application/graphql-response+json;charset=utf-8
- Reference
  - https://graphql.org/learn/serving-over-http/