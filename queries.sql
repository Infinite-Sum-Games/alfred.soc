-- name: RepositoryExistsQuery :one
SELECT EXISTS (
  SELECT 1 FROM repository WHERE id = $1
) AS found;

-- name: ParticipantExistsQuery :one
SELECT EXISTS (
  SELECT 1 FROM user_account 
  WHERE ghUsername = $1
  AND status = true
) AS found;

-- name: IsMaintainerOfRepositoryQuery :one
SELECT EXISTS (
  SELECT 1 FROM repository
  WHERE id = $1
    AND $2 = ANY(maintainers)
) AS is_maintainer;

-- name: UpdateRepositoryOnboardedQuery :one
UPDATE repository 
  SET onboarded = true
  WHERE url = $1
RETURNING name;