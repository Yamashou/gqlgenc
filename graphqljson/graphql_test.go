package graphqljson_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/Yamashou/gqlgenc/v3/graphqljson"
)

func TestUnmarshalGraphQL(t *testing.T) {
	t.Parallel()
	/*
		query {
			me {
				name
				height
			}
		}
	*/
	type query struct {
		Me struct {
			Name   string
			Height float64
		}
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"me": {
			"name": "Luke Skywalker",
			"height": 1.72
		}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	var want query
	want.Me.Name = "Luke Skywalker"
	want.Me.Height = 1.72

	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_graphqlTag(t *testing.T) {
	t.Parallel()

	type query struct {
		Foo string `graphql:"baz"`
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"baz": "bar"
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := query{
		Foo: "bar",
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_jsonTag(t *testing.T) {
	t.Parallel()

	type query struct {
		Foo string `json:"baz"`
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"foo": "bar"
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := query{
		Foo: "bar",
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_array(t *testing.T) {
	t.Parallel()

	type query struct {
		Foo []string
		Bar []string
		Baz []string
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"foo": [
			"bar",
			"baz"
		],
		"bar": [],
		"baz": null
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := query{
		Foo: []string{"bar", "baz"},
		Bar: []string{},
		Baz: []string(nil),
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

// When unmarshaling into an array, its initial value should be overwritten
// (rather than appended to).
func TestUnmarshalGraphQL_arrayReset(t *testing.T) {
	t.Parallel()

	got := []string{"initial"}

	err := graphqljson.UnmarshalData([]byte(`["bar", "baz"]`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"bar", "baz"}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_objectArray(t *testing.T) {
	t.Parallel()

	type query struct {
		Foo []struct {
			Name string
		}
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"foo": [
			{"name": "bar"},
			{"name": "baz"}
		]
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := query{
		Foo: []struct{ Name string }{
			{"bar"},
			{"baz"},
		},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_pointer(t *testing.T) {
	t.Parallel()

	type query struct {
		Foo *string
		Bar *string
	}

	var got query
	got.Bar = new(string) // Test that got.Bar gets set to nil.

	err := graphqljson.UnmarshalData([]byte(`{
		"foo": "foo",
		"bar": null
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	foo := "foo"

	want := query{
		Foo: &foo,
		Bar: nil,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_objectPointerArray(t *testing.T) {
	t.Parallel()

	type query struct {
		Foo []*struct {
			Name string
		}
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"foo": [
			{"name": "bar"},
			null,
			{"name": "baz"}
		]
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := query{
		Foo: []*struct{ Name string }{
			{"bar"},
			nil,
			{"baz"},
		},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_multipleFragment(t *testing.T) {
	t.Parallel()

	type UserFragment1 struct {
		Name string `json:"name"`
	}

	type UserFragment2 struct {
		Name string `json:"name"`
		User struct {
			Name string `json:"name"`
		} `graphql:"... on User"`
	}

	type query struct {
		Name string `json:"name"`
		UserFragment1
		UserFragment2
		User struct {
			Name string `json:"name"`
			UserFragment1
			UserFragment2
		} `graphql:"... on User"`
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{ "name": "John Doe" }`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := query{
		Name:          "John Doe",
		UserFragment1: UserFragment1{Name: "John Doe"},
		UserFragment2: UserFragment2{
			Name: "John Doe",
			User: struct {
				Name string `json:"name"`
			}{Name: "John Doe"},
		},
		User: struct {
			Name string `json:"name"`
			UserFragment1
			UserFragment2
		}{
			Name:          "John Doe",
			UserFragment1: UserFragment1{Name: "John Doe"},
			UserFragment2: UserFragment2{
				Name: "John Doe",
				User: struct {
					Name string `json:"name"`
				}{Name: "John Doe"},
			},
		},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_pointerWithInlineFragment(t *testing.T) {
	t.Parallel()

	type actor struct {
		User struct {
			DatabaseID uint64
		} `graphql:"... on User"`
		Login string
	}

	type query struct {
		Author actor
		Editor *actor
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"author": {
			"databaseId": 1,
			"login": "test1"
		},
		"editor": {
			"databaseId": 2,
			"login": "test2"
		}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	var want query
	want.Author = actor{
		User:  struct{ DatabaseID uint64 }{1},
		Login: "test1",
	}
	want.Editor = &actor{
		User:  struct{ DatabaseID uint64 }{2},
		Login: "test2",
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_unexportedField(t *testing.T) {
	t.Parallel()

	type query struct {
		//nolint:unused // This is a test.
		foo string
	}

	err := graphqljson.UnmarshalData([]byte(`{"foo": "bar"}`), new(query))
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}

	got := err.Error()
	want := ": : struct field for \"foo\" doesn't exist in any of 1 places to unmarshal"

	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_multipleValues(t *testing.T) {
	t.Parallel()

	type query struct {
		Foo string
	}

	err := graphqljson.UnmarshalData([]byte(`{"foo": "bar"}{"foo": "baz"}`), new(query))
	if err == nil {
		t.Fatal("got error: nil, want: non-nil")
	}

	if got, want := err.Error(), "invalid token '{' after top-level value"; got != want {
		t.Errorf("got error: %v, want: %v", got, want)
	}
}

func TestUnmarshalGraphQL_union(t *testing.T) {
	t.Parallel()
	/*
		{
			__typename
			... on ClosedEvent {
				createdAt
				actor {login}
			}
			... on ReopenedEvent {
				createdAt
				actor {login}
			}
		}
	*/
	type actor struct{ Login string }

	type reopenedEvent struct {
		Actor     actor
		CreatedAt time.Time
	}

	type issueTimelineItem struct {
		Typename    string `graphql:"__typename"`
		ClosedEvent struct {
			Actor     actor
			UpdatedAt time.Time
		} `graphql:"... on ClosedEvent"`
		ReopenedEvent reopenedEvent `graphql:"... on ReopenedEvent"`
	}

	var got issueTimelineItem

	err := graphqljson.UnmarshalData([]byte(`{
		"__typename": "ClosedEvent",
		"createdAt": "2017-06-29T04:12:01Z",
		"updatedAt": "2017-06-29T04:12:01Z",
		"actor": {
			"login": "shurcooL-test"
		}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := issueTimelineItem{
		Typename: "ClosedEvent",
		ClosedEvent: struct {
			Actor     actor
			UpdatedAt time.Time
		}{
			Actor: actor{
				Login: "shurcooL-test",
			},
			UpdatedAt: time.Unix(1498709521, 0).UTC(),
		},
		ReopenedEvent: reopenedEvent{
			Actor: actor{
				Login: "shurcooL-test",
			},
			CreatedAt: time.Unix(1498709521, 0).UTC(),
		},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_union2(t *testing.T) {
	t.Parallel()

	type SubscriptionItemFragment struct {
		ID string
	}

	type PurchaseItemFragment struct {
		ID string
	}

	type OrderFragment struct {
		SubscriptionItemOrder struct {
			SubscriptionItem SubscriptionItemFragment
		} `graphql:"... on SubscriptionItemOrder"`
		PurchaseItemFragment struct {
			PurchaseItem PurchaseItemFragment
		} `graphql:"... on PurchaseItemOrder"`
	}

	type BuyDashItemPayload struct {
		Order OrderFragment
	}

	var got BuyDashItemPayload

	resp := `
	{
		"order": {
			"subscriptionItem": {
				"id": "subscriptionItemOrderID"
			}
		}
	}
`

	err := graphqljson.UnmarshalData([]byte(resp), &got)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	want := BuyDashItemPayload{Order: OrderFragment{
		SubscriptionItemOrder: struct {
			SubscriptionItem SubscriptionItemFragment
		}{
			SubscriptionItem: SubscriptionItemFragment{ID: "subscriptionItemOrderID"},
		},
		PurchaseItemFragment: struct {
			PurchaseItem PurchaseItemFragment
		}{},
	}}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

// Issue https://github.com/shurcooL/githubv4/issues/18.
func TestUnmarshalGraphQL_arrayInsideInlineFragment(t *testing.T) {
	t.Parallel()
	/*
		query {
			search(type: ISSUE, first: 1, query: "type:pr repo:owner/name") {
				nodes {
					... on PullRequest {
						commits(last: 1) {
							nodes {
								url
							}
						}
					}
				}
			}
		}
	*/
	type query struct {
		Search struct {
			Nodes []struct {
				PullRequest struct {
					Commits struct {
						Nodes []struct {
							URL string `graphql:"url"`
						}
					} `graphql:"commits(last: 1)"`
				} `graphql:"... on PullRequest"`
			}
		} `graphql:"search(type: ISSUE, first: 1, query: \"type:pr repo:owner/name\")"`
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"search": {
			"nodes": [
				{
					"commits": {
						"nodes": [
							{
								"url": "https://example.org/commit/49e1"
							}
						]
					}
				}
			]
		}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	var want query
	want.Search.Nodes = make([]struct {
		PullRequest struct {
			Commits struct {
				Nodes []struct {
					URL string `graphql:"url"`
				}
			} `graphql:"commits(last: 1)"`
		} `graphql:"... on PullRequest"`
	}, 1)
	want.Search.Nodes[0].PullRequest.Commits.Nodes = make([]struct {
		URL string `graphql:"url"`
	}, 1)
	want.Search.Nodes[0].PullRequest.Commits.Nodes[0].URL = "https://example.org/commit/49e1"

	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_jsonRawMessage(t *testing.T) {
	t.Parallel()

	type query struct {
		JSONBlob    json.RawMessage `json:"jsonBlob"`
		JSONArray   json.RawMessage `json:"jsonArray"`
		JSONNumber  json.RawMessage `json:"jsonNumber"`
		JSONString  json.RawMessage `json:"jsonString"`
		JSONOmmited json.RawMessage `json:"jsonOmmited"`
		Number      int             `json:"number"`
		String      string          `json:"string"`
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"jsonBlob": {
			"foo": "bar"
		},
		"jsonArray": [1, "two", 3],
		"jsonNumber": 2,
		"jsonString": "json string",
		"number": 1,
		"string": "normal string"
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := query{
		JSONBlob:    []byte(`{"foo":"bar"}`),
		JSONArray:   []byte(`[1,"two",3]`),
		JSONNumber:  []byte(`2`),
		JSONString:  []byte(`"json string"`),
		JSONOmmited: nil,
		Number:      1,
		String:      "normal string",
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_jsonRawMessageInFragment(t *testing.T) {
	t.Parallel()

	type Object struct {
		Properties struct {
			ID       string
			Metadata json.RawMessage
		} `graphql:"... on Properties"`
		Value string
	}

	type query struct {
		Object         Object
		OptionalObject *Object
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"object": {
			"id": "81beda46-02c1-4641-aa7b-09cc6634c783",
			"metadata": {
				"created": "2021-05-03T21:27:28+00:00"
			},
			"value": "object value 1"
		},
		"optionalObject": {
			"id": "6f8af214-f307-4d4d-89d3-965d8b79e3bf",
			"metadata": {
				"created": "2021-05-03T21:27:28+00:00",
				"deleted": "2021-05-04T21:27:28+00:00"
			},
			"value": "object value 2"
		}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	var want query
	want.Object = Object{
		Properties: struct {
			ID       string
			Metadata json.RawMessage
		}{
			ID:       "81beda46-02c1-4641-aa7b-09cc6634c783",
			Metadata: []byte(`{"created":"2021-05-03T21:27:28+00:00"}`),
		},
		Value: "object value 1",
	}
	want.OptionalObject = &Object{
		Properties: struct {
			ID       string
			Metadata json.RawMessage
		}{
			ID:       "6f8af214-f307-4d4d-89d3-965d8b79e3bf",
			Metadata: []byte(`{"created":"2021-05-03T21:27:28+00:00","deleted":"2021-05-04T21:27:28+00:00"}`),
		},
		Value: "object value 2",
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGraphQL_map(t *testing.T) {
	t.Parallel()

	type query struct {
		Outputs map[string]any
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
			"outputs":{
                                 "vpc":"1",
                                 "worker_role_arn":"2"
        	}
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := query{
		map[string]any{
			"vpc":             "1",
			"worker_role_arn": "2",
		},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

type Number int64

const (
	NumberOne Number = 1
	NumberTwo Number = 2
)

func (n *Number) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return errors.New("enums must be strings")
	}

	switch str {
	case "ONE":
		*n = NumberOne
	case "TWO":
		*n = NumberTwo
	default:
		return fmt.Errorf("number not found Type: %d", n)
	}

	return nil
}

func TestUnmarshalGQL(t *testing.T) {
	t.Parallel()

	type query struct {
		Enum Number
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"enum": "ONE"
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := query{
		Enum: NumberOne,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGQL_array(t *testing.T) {
	t.Parallel()

	type query struct {
		Enums []Number
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"enums": ["ONE", "TWO"]
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := query{
		Enums: []Number{NumberOne, NumberTwo},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGQL_pointer(t *testing.T) {
	t.Parallel()

	type query struct {
		Enum *Number
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"enum": "ONE"
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	v := NumberOne

	want := query{
		Enum: &v,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGQL_pointerArray(t *testing.T) {
	t.Parallel()

	type query struct {
		Enums []*Number
	}

	var got query

	err := graphqljson.UnmarshalData([]byte(`{
		"enums": ["ONE", "TWO"]
	}`), &got)
	if err != nil {
		t.Fatal(err)
	}

	one := NumberOne
	two := NumberTwo

	want := query{
		Enums: []*Number{&one, &two},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestUnmarshalGQL_pointerArrayReset(t *testing.T) {
	t.Parallel()

	got := []*Number{new(Number)}

	err := graphqljson.UnmarshalData([]byte(`["TWO"]`), &got)
	if err != nil {
		t.Fatal(err)
	}

	want := []*Number{new(Number)}
	*want[0] = NumberTwo

	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}
