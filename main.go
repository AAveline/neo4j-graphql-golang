package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/graphql-go/graphql"
	bolt "github.com/johnnadratowski/golang-neo4j-bolt-driver"
)

type user struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

var data map[string]user

var userType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
		},
	},
)

type person struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

var persons []person

var activityType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Activity",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
		},
	},
)

var personType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Person",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
			},
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"role": &graphql.Field{
				Type: graphql.String,
			},
			"activity": &graphql.Field{
				Type: activityType,
			},
		},
	},
)

var queryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"person": &graphql.Field{
				Type: personType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					idQuery, _ := p.Args["id"].(string)
					id, _ := strconv.Atoi(idQuery)
					conn, err := bolt.NewDriver().OpenNeo("bolt://neo4j:neo4jbis@172.17.0.2:7687")
					if err != nil {
						panic(err)
					}

					defer conn.Close()

					stmt, _ := conn.PrepareNeo(`
						MATCH (n:Person {id: {id}})
						OPTIONAL MATCH (n)-[:HAS_ACTIVITY]->(a:Activity)
						WITH n, a { .* } AS activity
						RETURN n { .*, activity: activity}
					`)

					result, err := stmt.QueryNeo(map[string]interface{}{"id": id})

					if err != nil {
						fmt.Println(err)
					}

					data, _, rowsErr := result.NextNeo()

					if rowsErr != nil {
						fmt.Println(rowsErr)
					}

					return data[0].(map[string]interface{}), nil
				},
			},
			"persons": &graphql.Field{
				Type: graphql.NewList(personType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					conn, err := bolt.NewDriver().OpenNeo("bolt://neo4j:neo4jbis@172.17.0.2:7687")
					if err != nil {
						panic(err)
					}

					defer conn.Close()

					stmt, err := conn.PrepareNeo(`
					MATCH (n:Person)
					OPTIONAL MATCH (n)-[:HAS_ACTIVITY]->(a:Activity)
					WITH n, a { .* } AS activity
					RETURN n { .*, activity: activity}
					`)

					if err != nil {
						fmt.Println(err)
					}

					rows, err := stmt.QueryNeo(nil)

					if err != nil {
						fmt.Println(err)
					}
					var results []interface{}

					result, _, rowsErr := rows.All()
					for _, value := range result {

						results = append(results, value[0])
					}

					if rowsErr != nil {
						fmt.Println(err)
					}

					return results, nil
				},
			},
		},
	})

var schema, _ = graphql.NewSchema(
	graphql.SchemaConfig{
		Query: queryType,
	},
)

func executeQuery(query string, schema graphql.Schema) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})
	if len(result.Errors) > 0 {
		fmt.Printf("wrong result, unexpected errors: %v", result.Errors)
	}
	return result
}

func main() {
	_ = importJSONDataFromFile("data.json", &data)

	http.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		result := executeQuery(r.URL.Query().Get("query"), schema)
		json.NewEncoder(w).Encode(result)
	})

	fmt.Println("Now server is running on port 8080")
	fmt.Println("Test with Get      : curl -g 'http://localhost:8080/graphql?query={person(id:\"1\"){name}}'")
	http.ListenAndServe(":8080", nil)
}

//Helper function to import json from file to map
func importJSONDataFromFile(fileName string, result interface{}) (isOK bool) {
	isOK = true
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Print("Error:", err)
		isOK = false
	}
	err = json.Unmarshal(content, result)
	if err != nil {
		isOK = false
		fmt.Print("Error:", err)
	}
	return
}
