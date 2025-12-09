package main

import (
	"appaws/repository/schema"
	"appaws/repository/schema/seeds"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	. "github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/images/localstack"
	"github.com/go-andiamo/marrow/images/mysql"
	"github.com/go-andiamo/marrow/with"
	"os"
	"path/filepath"
)

const (
	nonExistentId = Var("non-uuid")
	apiLogs       = Var("api-logs")
)

var endpoints = []Endpoint_{
	Endpoint("/api", "Root",
		Method(GET, "Get root").
			AssertOK().
			AssertEqual(JsonPath(Body, "hello"), "world").
			// only try to read api logs if running api as image...
			If(After, DefaultVar(apiLogs, false), ExpectGreaterThan(Len(ApiLogs(-1)), 0)),
		Endpoint("/pets", "Pets",
			Method(GET, "Get pets (empty)").
				AssertOK().
				AssertLen(Body, 0),
			Method(POST, "Create pet").
				SetVar(Before, "msgs-count-before", localstack.SNSMessagesCount("")).
				RequestBody(JSON{
					"name": "Felix",
					"dob":  "2025-11-01",
					"category": JSON{
						"id": Query("", "SELECT id FROM categories"),
					},
				}).
				AssertCreated().
				AssertOnlyHasProperties(Body, "id", "name", "dob", "category", "$ref").
				SetVar(After, "created-pet-id", JsonPath(Body, "id")).
				Wait(After, 250). // wait for SNS messages to propagate
				AssertGreaterThan(localstack.SNSMessagesCount(""), Var("msgs-count-before")),
			Method(GET, "Get pets (non-empty)").
				AssertOK().
				AssertLen(Body, 1),
			Endpoint("/{petId}", "Pet",
				Method(GET, "Get pet (not found)").
					PathParam(nonExistentId).
					AssertNotFound(),
				Method(GET, "Get pet").
					PathParam(Var("created-pet-id")).
					AssertOK().
					AssertOnlyHasProperties(Body, "id", "name", "dob", "category", "$ref"),
				Method(DELETE, "Delete pet (not found)").
					PathParam(nonExistentId).
					AssertNotFound(),
				Method(DELETE, "Delete pet successful").
					SetVar(Before, "before-count", Query("", "SELECT COUNT(*) FROM pets")).
					PathParam(Var("created-pet-id")).
					AssertNoContent().
					AssertGreaterThan(Var("before-count"), Query("", "SELECT COUNT(*) FROM pets")),
			),
		),
		Endpoint("/categories", "Categories",
			Method(GET, "Get categories").
				AssertOK().
				AssertGreaterThan(JsonPath(Body, LEN), 0),
			Endpoint("/{categoryId}", "Category",
				Method(GET, "Get category (not found)").
					SetVar(Before, "categoryId", Query("", "SELECT id FROM categories")).
					PathParam(nonExistentId).
					AssertNotFound(),
				Method(GET, "Get category (found)").
					PathParam(Var("categoryId")).
					AssertOK().
					AssertOnlyHasProperties(Body, "id", "name", "$ref"),
			),
		),
	),
}

func main() {
	apiEnv := map[string]any{
		"AWS_ACCESS_KEY_ID":     "{$svc:aws:accesskey}",
		"AWS_SECRET_ACCESS_KEY": "{$svc:aws:secretkey}",
		"AWS_REGION":            "{$svc:aws:region}",
		"AWS_ENDPOINT_URL":      "http://host.docker.internal:{$svc:aws:mport}",
		//"AWS_SESSION_TOKEN": "{$svc:ssm:sessiontoken}",
	}
	s := Suite(endpoints...)
	s = s.Init(
		with.DisableReaperShutdowns(true),
		// tell the tests we want to read api logs...
		with.Var(string(apiLogs), true),
		with.Var(string(nonExistentId), "00000000-0000-485c-0000-000000000000"),
		with.Make(with.Supporting, absPath("./Makefile"), 0, false),
		with.ApiImage("petstore", "latest", 8080, apiEnv, false),
		mysql.With("mysql", mysql.Options{
			Database: "petstore",
			//LeaveRunning: true,
			Migrations: []mysql.Migration{
				{
					Filesystem: schema.Migrations,
				},
				{
					Filesystem: seeds.Migrations,
					TableName:  "schema_migrations_seeds",
				},
			}}),
		localstack.With(localstack.Options{
			//LeaveRunning: true,
			Services: localstack.Services{localstack.All},
			SNS: localstack.SNSOptions{
				CreateTopics: []sns.CreateTopicInput{
					{
						Name: aws.String("pets"),
					},
				},
				TopicsSubscribe: true,
			},
			SecretsManager: localstack.SecretsManagerOptions{
				Secrets: map[string]any{
					"db_username": TemplateString("{$svc:mysql:username}"),
					"db_password": TemplateString("{$svc:mysql:password}"),
				},
			},
			SSM: localstack.SSMOptions{
				Prefix: "/app/petstore",
				InitialParams: map[string]any{
					"api_port":       "8080",
					"pets_topic_arn": TemplateString("{$svc:sns:arn:pets}"),
					"db_host":        "host.docker.internal",
					"db_port":        TemplateString("{$svc:mysql:mport}"),
					"db_name":        "petstore",
				},
			},
			//Dynamo:              localstack.DynamoOptions{},
			//S3:                  localstack.S3Options{},
			//SQS:                 localstack.SQSOptions{},
			//Lambda:              localstack.LambdaOptions{},
		}))
	err := s.Run()
	if err != nil {
		panic(err)
	}
}

func absPath(path string) string {
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Join(cwd, path)
		}
	}
	return path
}
