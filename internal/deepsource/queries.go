package deepsource

// GraphQL queries for DeepSource API

const GetRepositoryIssuesQuery = `
query GetRepositoryIssues($owner: String!, $name: String!, $provider: VCSProvider!, $first: Int, $after: String) {
  repository(login: $owner, name: $name, vcsProvider: $provider) {
    issues(first: $first, after: $after) {
      edges {
        node {
          id
          issue {
            shortcode
            title
            category
            severity
            description
            analyzer {
              shortcode
            }
          }
          occurrences(first: 100) {
            edges {
              node {
                id
                path
                beginLine
                endLine
              }
            }
          }
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
`

const GetIssueDetailQuery = `
query GetIssueDetail($issueId: ID!) {
  issue(id: $issueId) {
    id
    title
    shortcode
    category
    issue {
      severity
      description
      recommendation
    }
    occurrences(first: 1) {
      edges {
        node {
          id
          path
          beginLine
          endLine
          beginColumn
          endColumn
        }
      }
    }
  }
}
`
