-- name: ParticipantExistsQuery :one
SELECT EXISTS (
  SELECT 1 FROM user_account 
  WHERE ghUsername = $1
  AND status = true
) AS found;

-- name: CheckForMaintainerQuery :one
SELECT EXISTS (
  SELECT 1 FROM repository
  WHERE url = $1
    AND $2 = ANY(maintainers)
) AS is_maintainer;

-- name: VerifyRepository :one
UPDATE repository 
  SET installation_id = $1
  WHERE url = $2
RETURNING name;

-- name: CheckOpenIssueQuery :one
SELECT EXISTS(
  SELECT 1 FROM issues
  WHERE url = $1
) AS found;

-- name: AddNewIssueQuery :exec
INSERT INTO issues (id, title, repoUrl, url)
VALUES ($1, $2, $3, $4);

-- name: UpdateIssueDifficultyQuery :one
UPDATE issues
SET 
  difficulty = $1
WHERE url = $2
RETURNING url;

-- name: CheckIfTagExistInIssueQuery :one
SELECT EXISTS (
  SELECT 1
  FROM issues
  WHERE tags @> ARRAY[$1]::text[]
  AND url = $2
) AS tag_exists;

-- name: AddIssueTagQuery :one
UPDATE issues
SET tags = array_append(tags, $1),
    updated_at = NOW()
WHERE url = $2
AND NOT tags @> ARRAY[$1]::text[]
RETURNING tags;

-- name: IssueAssignQuery :exec
INSERT INTO issue_claims (
    ghUsername,
    issue_url,
    claimed_on,
    elapsed_on
) VALUES (
    $1,
    $2,
    $3,
    $4
);

-- name: IssueUnassignQuery :one
DELETE FROM issue_claims
WHERE
    ghUsername = $1 AND issue_url = $2 AND elapsed_on > NOW()
RETURNING ghUsername;

-- name: CloseIssueQuery :one
UPDATE issues
SET
    resolved = TRUE,
    updated_at = NOW()
WHERE
    url = $1
RETURNING url;

-- name: OpenIssueQuery :one
UPDATE issues
SET
    resolved = FALSE,
    updated_at = NOW()
WHERE
    url = $1
RETURNING url;

-- name: ExtendClaimQuery :one
UPDATE issue_claims
SET
    elapsed_on = elapsed_on + make_interval(days => $1)
WHERE
    ghUsername = $2 
    AND issue_url = $3
    AND elapsed_on > NOW()
returning ghUsername;
