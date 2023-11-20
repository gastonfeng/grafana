Hi <%= owner %>!

A while back to changed to prefer using `data-testid` for E2E selectors instead of `aria-label`. There's only a few of these old `aria-label`-based selectors left, so we've made this list of files that your team owns so we can tidy them up 🙌.

For some guidance on how exactly to fix these, check out [this comment](https://github.com/grafana/grafana/issues/36523#issuecomment-1819160512).

There are <%= totalIssueCount %> <%= plural('issue', totalIssueCount) %> over <%= fileCount %> <%= plural('file', fileCount) %>:
<% files.forEach((file) => { %>
- [ ] <%= file.issueCount %> <%= plural('issue', file.issueCount) %> in `<%= file.fileName %>` <% }) %>
