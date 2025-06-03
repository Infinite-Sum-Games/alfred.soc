# Alfred - Webhook Butler for Season of Code

## Overview
Alfred is your trusty manager of all different webhook events that are sent 
via GitHub. Currently, it handles the follows:

1. **Issue Management:**Creation of new-issues through adding of the appropriate
labels as well as inclusion of new features and bug-report events.
2. **Pull Request Management:**Whenever new pull requests are opened, tagged and
merged, webhook events are captured and sent to appropriate handlers. 
3. **GitHub Repository Management:**Onboarding new repositories and maintainers 
by admins - [Ritesh Koushik](https://github.com/IAmRiteshKoushik) and [Ashwin Narayanan](https://github.com/Ashrockzzz2003)
4. **Verifying Bot Commands:**All bot commands will be verified and then streamed
to be handled by the [github-bot](https://github.com/Infinite-Sum-Games/devpool.soc)
5. **LIVE Stream Updates:**Directly dropping events to LIVE "valkey-stream" to 
be picked up and sent by SSE handler at [api-server](https://github.com/Infinite-Sum-Games/pulse.soc).

## Authors
This project has been authored and tested by [Ritesh Koushik](https://github.com/IAmRiteshKoushik)