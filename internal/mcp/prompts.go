package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// prompt is one prompt definition with the text it expands to.
type prompt struct {
	def  *sdk.Prompt
	text func(args map[string]string) string
}

// prompts are the one-command workflows canon offers.
var prompts = []prompt{
	{
		def: &sdk.Prompt{Name: "canon-review", Title: "Review this branch against the canon",
			Description: "Runs review on the current branch and writes a PR-ready summary",
			Arguments:   []*sdk.PromptArgument{{Name: "base", Description: "Branch to compare against (default origin/main)"}}},
		text: func(a map[string]string) string {
			base := a["base"]
			if base == "" {
				base = "origin/main"
			}
			return fmt.Sprintf(`Review the current branch against the canon.

1. Call the canon review tool with base %q.
2. For every error finding, fix it or explain why it is an accepted exception, quoting the record's Exceptions section.
3. For each concern, call explain with its record and decide whether the change follows it.
4. Write a PR-ready summary: what changed, the findings fixed, the concerns and how each is addressed, and the records touched. Quote each finding's "why" rather than restating the rule.
Do not invent standards: if something is not in the canon, say so.`, base)
		},
	},
	{
		def: &sdk.Prompt{Name: "canon-new-record", Title: "Draft a new decision record",
			Description: "Interviews you about a decision and drafts a record in the right format",
			Arguments:   []*sdk.PromptArgument{{Name: "topic", Description: "What the decision is about"}}},
		text: func(a map[string]string) string {
			topic := a["topic"]
			if topic == "" {
				topic = "a decision I just made"
			}
			return fmt.Sprintf(`Help me write a canon decision record about %s.

First read canon://records and call decide with the topic, so we do not duplicate or contradict an existing record; if one exists, propose superseding or amending it instead.
Then ask me, one question at a time: the context (what problem or incident led here), the decision (what we do, stated as an instruction), the consequences (what it costs and what it enables), exceptions, and a good and a bad example.
Draft the record as Markdown with this frontmatter: id (kebab-case, starting with its scope), title, summary (one sentence, used as "why" in findings), status: proposed, supersedes, scope, applies_to.files, tags, rules (rule IDs, or [] if nothing checks it yet), strength (must, should or prefer), updated (today).
Body sections, in order: ## Context, ## Decision, ## Consequences, ## Exceptions, ## Examples.
Finally, suggest whether a linter rule or a built-in check could enforce it, and say which file under canon/records/<scope>/ it belongs in.`, topic)
		},
	},
	{
		def: &sdk.Prompt{Name: "canon-adopt", Title: "Adopt the canon in an existing repo",
			Description: "Runs profile + check on an existing repo and drafts a remediation plan by severity",
			Arguments:   []*sdk.PromptArgument{{Name: "path", Description: "Repo path (default .)"}}},
		text: func(a map[string]string) string {
			p := a["path"]
			if p == "" {
				p = "."
			}
			return fmt.Sprintf(`Plan how to bring the repo at %q in line with the canon.

1. Call profile with path %q. Note the profile, the records in force, and any checkers that are missing locally; list how to install those first.
2. Call check with min_severity "suggestion".
3. Group findings by record, then order the groups by severity (errors first) and by how many findings each has.
4. Draft a remediation plan: one step per record, with the files affected, the fix (use each finding's fix when it has one), and an estimate of effort (small, medium, large). Mark steps that can be done mechanically.
5. End with the records that have no automated rule but apply to this profile, as a short checklist for a human to confirm.`, p, p)
		},
	},
}

// addPrompts registers the prompts.
func (s *Server) addPrompts(srv *sdk.Server) {
	for _, p := range prompts {
		srv.AddPrompt(p.def, func(_ context.Context, req *sdk.GetPromptRequest) (*sdk.GetPromptResult, error) {
			return &sdk.GetPromptResult{
				Description: p.def.Description,
				Messages:    []*sdk.PromptMessage{{Role: "user", Content: &sdk.TextContent{Text: p.text(req.Params.Arguments)}}},
			}, nil
		})
	}
}
