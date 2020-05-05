# Contributing to <project>


---

**All communication on GitHub, the community forum, and other HashiCorp-provided communication channels is subject to [the HashiCorp community guidelines](https://www.hashicorp.com/community-guidelines).**

This document provides guidance on our recommended practices for contribution. It covers what we're looking for in order to help set some expectations and help you get the most out of participation in this project. 

To record a bug report, enhancement proposal, or give any other product feedback, please [open a GitHub issue](https://github.com/hashicorp/<project>/issues/new/choose) using the most appropriate issue template. Please do fill in all of the information the issue templates request, because we've seen from experience that this will maximize the chance that we'll be able to act on your feedback.

---

<!-- MarkdownTOC autolink="true" -->

- [Contributing Fixes](#contributing-fixes)
- [Proposing a Change](#proposing-a-change)
- [Maintainers](#maintainers)
- [Pull Request Lifecycle](#pull-request-lifecycle)
  - [Getting Your Pull Requests Merged Faster](#getting-your-pull-requests-merged-faster)

<!-- /MarkdownTOC -->

## Proposing a Change

In order to be respectful of the time of community contributors, we aim to discuss potential changes in GitHub issues prior to implementation. That will allow us to give design feedback up front and set expectations about the scope of the change, and, for larger changes, how best to approach the work such that the team can review it and merge it along with other concurrent work.

If the bug you wish to fix or enhancement you wish to implement isn't already covered by a GitHub issue that contains feedback from the team, please do start a discussion (either in [a new GitHub issue](https://github.com/hashicorp/<project>/issues/new/choose) or an existing one, as appropriate) before you invest significant development time. If you mention your intent to implement the change described in your issue, the team can, as best as possible, prioritize including implementation-related feedback in the subsequent discussion.

For large proposals that could entail a significant design phase, we wish to be up front with potential contributors that, unfortunately, we are unlikely to be able to give prompt feedback. We are still interested to hear about your use-cases so that we can consider ways to meet them as part of other larger projects.

This repository is primarily maintained by a small team at HashiCorp along with their other responsibilities, so unfortunately we cannot always respond promptly to pull requests, particularly if they do not relate to an existing GitHub issue where the team has already participated and indicated willingness to work on the issue or accept PRs for the proposal. We *are* grateful for all contributions however, and will give feedback on pull requests as soon as we're able.

## Maintainers

Maintainers are key contributors to our Open Source project. They contribute their time and expertise and we ask that the community take extra special care to be mindful of this when interacting with them.

There is no expectation on response time for our maintainers; they may be indisposed for prolonged periods of time. Please be patient. Discussions on when code becomes "unmaintained" will be on a case-by-case basis. 

## Pull Request Lifecycle

1. You are welcome to submit a [draft pull request](https://github.blog/2019-02-14-introducing-draft-pull-requests/) for commentary or review before it is fully completed. It's also a good idea to include specific questions or items you'd like feedback on from other contributors. As maintainers, we will make a best effort attempt to participate, but cannot guarantee that we will be able to review in a timely manner.
2. Once you believe your pull request is ready to be merged you can create your pull request.
3. When time permits our team members will look over your contribution and either merge, or provide comments letting you know if there is anything left to do. It may take some time for us to respond. We may also have questions that we need answered about the code, either because something doesn't make sense to us or because we want to understand your thought process. We kindly ask that you do not target specific team members with review requests or comments. 
4. If we have requested changes, you can either make those changes or, if you disagree with the suggested changes, we can have a conversation about our reasoning and agree on a path forward. This may be a multi-step process. Our view is that pull requests are a chance to collaborate, and we welcome conversations about how to do things better. It is the contributor's responsibility to address any changes requested. While reviewers are happy to give guidance, it is unsustainable for us to perform the coding work necessary to get a PR into a mergeable state.
5. Once all outstanding comments and checklist items have been addressed, your contribution will be merged! Merged PRs may or may not be included in the next release based on changes the teams deems as breaking or not. The team takes care of updating the [CHANGELOG.md](https://github.com/hashicorp/<project>/blob/master/CHANGELOG.md) as they merge.
6. In some cases, we might decide that a PR should be closed without merging. We'll make sure to provide clear reasoning when this happens. Following the recommended process above is one of the ways to ensure you don't spend time on a PR we can't or won't merge.

### Getting Your Pull Requests Merged Faster

It is much easier to review pull requests that are:

1. Well-documented: Try to explain in the pull request comments what your change does, why you have made the change, and provide instructions for how to produce the new behavior introduced in the pull request. If you can, provide screen captures or terminal output to show what the changes look like. This helps the reviewers understand and test the change.
2. Small: Try to only make one change per pull request. If you found two bugs and want to fix them both, that's *awesome*, but it's still best to submit the fixes as separate pull requests. This makes it much easier for reviewers to keep in their heads all of the implications of individual code changes, and that means the PR takes less effort and energy to merge. In general, the smaller the pull request, the sooner reviewers will be able to make time to review it.

If we request changes, try to make those changes in a timely manner. Otherwise, PRs can go stale and be a lot more work for all of us to merge in the future.

Even with everyone making their best effort to be responsive, it can be time-consuming to get a PR merged. It can be frustrating to deal with the back-and-forth as we make sure that we understand the changes fully. Please bear with us, and please know that we appreciate the time and energy you put into the project.