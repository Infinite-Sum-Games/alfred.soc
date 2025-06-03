CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Maintainers table
CREATE TABLE IF NOT EXISTS maintainers(
  id SERIAL NOT NULL,
  ghUsername TEXT NOT NULL UNIQUE,
  full_name TEXT NOT NULL,

  CONSTRAINT "maintainers_pkey" PRIMARY KEY (id)
);

-- UserAccounts table
CREATE TABLE IF NOT EXISTS user_account(
  id SERIAL NOT NULL,
  email TEXT NOT NULL,
  ghUsername TEXT NOT NULL UNIQUE,
  status BOOLEAN DEFAULT true,
  bounty INT NOT NULL DEFAULT 0,
  refresh_token TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  CONSTRAINT "user_account_pkey" PRIMARY KEY (id)
);

-- GitHub repositories table
CREATE TABLE IF NOT EXISTS repository(
  id UUID NOT NULL,
  name TEXT NOT NULL,
  description TEXT NOT NULL,
  url TEXT NOT NULL,
  tags TEXT[],
  maintainers TEXT[],
  onboarded BOOLEAN DEFAULT false,
  is_internal BOOLEAN DEFAULT false,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  CONSTRAINT "repository_pkey" PRIMARY KEY (id)
);

-- Issues for repositories table
CREATE TABLE IF NOT EXISTS issues(
  id UUID NOT NULL,
  title TEXT NOT NULL,
  repoId UUID NOT NULL,
  url TEXT NOT NULL UNIQUE,
  resolved BOOLEAN DEFAULT false,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  CONSTRAINT "issues_pkey" PRIMARY KEY (id),
  CONSTRAINT "issues_repoid_fkey" 
    FOREIGN KEY (repoId)
      REFERENCES repository(id)
        ON DELETE RESTRICT
        ON UPDATE CASCADE
); 

-- Claims table
CREATE TABLE IF NOT EXISTS issue_claims(
  id SERIAL NOT NULL,
  ghUsername TEXT NOT NULL,
  issue_id UUID NOT NULL,
  claimed_on TIMESTAMP NOT NULL,
  elapsed_on TIMESTAMP NOT NULL,

  CONSTRAINT "issue_claims_pkey" PRIMARY KEY (id),
  CONSTRAINT "issue_claims_ghUsername_fkey"
    FOREIGN KEY (ghUsername)
      REFERENCES user_account(ghUsername)
        ON DELETE RESTRICT
        ON UPDATE CASCADE,
  CONSTRAINT "issue_claims_issue_id_fkey"
    FOREIGN KEY (issue_id)
      REFERENCES issues(id)
        ON DELETE RESTRICT
        ON UPDATE CASCADE
);