package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/machinebox/graphql"
)

type RepositoryStuct struct {
	Repository struct {
		Refs struct {
			Edges []struct {
				Node struct {
					Name string
				}
				Cursor string
			}
			PageInfo struct {
				EndCursor       string
				HasNextPage     bool
				HasPreviousPage bool
				StartCursor     string
			}
		}
	}
}

type RepositoryTag struct {
	Name string
}

var (
	GITHUB_TOKEN     = os.Getenv("GH_SOURCE_PAT")
	GRAPHQL_ENDPOINT = "https://github.flexport.io/api/graphql"
	OWNER            = "flexport"
	REPOSITORY       = "flexport"
	ENTERPRISE_URL   = "https://github.flexport.io"
)

func main() {
	var responseData RepositoryStuct
	var tags []RepositoryTag
	responseData = CallGraphQLAPI(OWNER, REPOSITORY, "null")
	for _, tag := range responseData.Repository.Refs.Edges {
		tagObject := RepositoryTag{
			Name: tag.Node.Name,
		}
		tags = append(tags, tagObject)
	}
	for responseData.Repository.Refs.PageInfo.HasNextPage {
		responseData = CallGraphQLAPI(
			OWNER,
			REPOSITORY,
			responseData.Repository.Refs.PageInfo.EndCursor,
		)
		for _, tag := range responseData.Repository.Refs.Edges {
			tagObject := RepositoryTag{
				Name: tag.Node.Name,
			}
			tags = append(tags, tagObject)
		}
	}
	for _, tag := range tags {
		fmt.Println("Deleting Tag", tag.Name)
		deleteTag(tag.Name)
	}
}

func deleteTag(tag string) {
	apiURL := fmt.Sprintf(
		"%s/api/v3/repos/%s/%s/git/refs/tags/%s",
		ENTERPRISE_URL,
		OWNER,
		REPOSITORY,
		tag,
	)
	client := &http.Client{}
	request, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		fmt.Println("Unable to create request: ", err)
	}
	request.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Error sending request ", err)
	}
	defer response.Body.Close()
	fmt.Println(response.StatusCode)
	if response.StatusCode == http.StatusNoContent {
		fmt.Printf("%s tag has been deleted", tag)
	} else {
		fmt.Printf("Unable to delete tag %s", tag)
	}
}

func CallGraphQLAPI(owner string, repository string, cursor string) RepositoryStuct {
	if cursor != "null" {
		cursor = `"` + cursor + `"`
	}

	query := fmt.Sprintf(`query Repository {
    repository(owner: "%s", name: "%s") {
        refs(refPrefix: "refs/tags/", first: 100, after: %s) {
            totalCount
            edges {
                node {
                    id
                    name
                }
                cursor
            }
            pageInfo {
                endCursor
                hasNextPage
                hasPreviousPage
                startCursor
            }
        }
    }
}
`, owner, repository, cursor)
	var responseData RepositoryStuct
	client := graphql.NewClient(GRAPHQL_ENDPOINT)
	req := graphql.NewRequest(query)
	req.Header.Set("Cache-Control", "no-cache")
	authorization_header := "Bearer " + GITHUB_TOKEN
	req.Header.Set("Authorization", authorization_header)
	ctx := context.Background()
	if err := client.Run(ctx, req, &responseData); err != nil {
		fmt.Println("Error calling graphql api: ", err)
	}
	return responseData
}
