package main

import (
	"fmt"
	. "github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/with"
	"os"
	"petstore/api"
)

var endpoints = []Endpoint_{
	Endpoint("/api", "Root",
		Method(GET, "Get root").
			AssertOK().
			AssertEqual(JsonPath(Body, "hello"), "world"),
		Endpoint("/categories", "Categories",
			Method(GET, "Get first category id (used for creating pet)").
				RequireOK().
				RequireGreaterThan(JsonPath(Body, LEN), 0).
				SetVar(After, "category-id", JsonPath(JsonPath(Body, "0"), "id")),
		),
		Endpoint("/pets", "Pets",
			Method(GET, "Get pets (empty)").
				AssertOK().
				AssertLen(Body, 0),
			Method(POST, "Create pet").
				RequestBody(JSON{
					"name": "Felix",
					"dob":  "2025-11-01",
					"category": JSON{
						"id": Var("category-id"),
					},
				}).
				AssertCreated().
				SetVar(After, "created-pet-id", JsonPath(Body, "id")),
			Endpoint("/{petId}", "Pet",
				Method(GET, "Get pet (not found)").
					PathParam(Var("non-uuid")).
					AssertNotFound(),
				Method(PUT, "Update pet (not found)").
					PathParam(Var("non-uuid")).
					AssertNotFound(),
				Method(PUT, "Update pet successful").
					PathParam(Var("created-pet-id")).
					RequestBody(JSON{
						"name": "Feline",
						"dob":  "2025-11-02",
						"category": JSON{
							"id": Var("category-id"),
						},
					}).
					AssertOK(),
				Method(DELETE, "Delete pet (not found)").
					PathParam(Var("non-uuid")).
					AssertNotFound(),
				Method(DELETE, "Delete pet successful").
					PathParam(Var("created-pet-id")).
					AssertNoContent(),
			),
		),
		Endpoint("/categories", "Categories",
			Method(GET, "Get categories").
				AssertOK().
				AssertGreaterThan(JsonPath(Body, LEN), 0),
			Endpoint("/{categoryId}", "Category",
				Method(GET, "Get category (not found)").
					PathParam(Var("non-uuid")).
					AssertNotFound(),
				Method(GET, "Get category (found)").
					PathParam(Var("category-id")).
					AssertOK(),
			),
		),
	),
}

func main() {
	port := 8080

	// oas spec file...
	spec, err := os.Open("spec.yaml")
	if err != nil {
		panic(err)
	}
	defer spec.Close()

	// start the api we're testing...
	a := api.NewApi()
	go a.Start(port)

	// initialise the suite...
	var cov *coverage.Coverage
	s := Suite(endpoints...).Init(
		with.ApiHost("localhost", port),
		with.ReportCoverage(func(coverage *coverage.Coverage) {
			cov = coverage
		}),
		with.OAS(spec),
		with.Repeats(10, true),
		with.Logging(os.Stdout, os.Stdout),
		with.Var("non-uuid", "00000000-0000-485c-0000-000000000000"),
		with.TraceTimings(),
	)

	// run the suite...
	err = s.Run()
	if err != nil {
		panic(err)
	}
	specCov, err := cov.SpecCoverage()
	if err != nil {
		panic(err)
	}
	total, covered, perc := specCov.PathsCovered()
	fmt.Printf("\nPaths covered:\n\t  Total: %d\n\tCovered: %d\n\tPercent: %.1f%%\n", total, covered, perc*100)
	total, covered, perc = specCov.MethodsCovered()
	fmt.Printf("Methods covered:\n\t  Total: %d\n\tCovered: %d\n\tPercent: %.1f%%\n", total, covered, perc*100)

	stats, ok := cov.Timings.Stats(false)
	if !ok {
		panic("no timing stats")
	}
	fmt.Printf(`Timings:
	    Mean: %s
	 Std.Dev: %s
	Variance: %f
	 Minimum: %s
	 Maximum: %s
	     P50: %s
	     P90: %s
	     P99: %s
	   Count: %d`, stats.Mean, stats.StdDev, stats.Variance, stats.Minimum, stats.Maximum, stats.P50, stats.P90, stats.P99, stats.Count)
}
