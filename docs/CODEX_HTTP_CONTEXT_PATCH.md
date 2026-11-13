# Temporary Codex HTTP context compatibility patch

Authorized by Zeeland on 2026-11-14 for the customer LLM Route gateway. The
default remains source-read-only. This is a bounded, temporary exception for the
reported Codex 0.169 HTTP 400.

## Scope

The existing Pages fix normalizes two Desktop page-context formats. Codex 0.169
also emits standalone agent.browser_info, agent.goal_objective and
agent.untrusted_context with no call_id. The patch converts these three known,
textual external-context outputs into user messages using external_NAME markers.
It preserves ordering, text, previous response IDs, paired tool history and exact
numeric metadata. Nonblank call IDs, malformed/null IDs, unknown names, encrypted
array outputs and duplicate JSON members retain the existing behavior.

Implementation is confined to the context normalizer and its handler call site.
Focused regression tests cover all three names and existing Pages behavior.
Dynamic external-context names beyond the allowlist remain outside this patch.

## Retirement

The scheduled one-time review is 2026-11-21 10:00 Asia/Shanghai, Codex automation
sub2api-2. The owner authorizes adopting a formal upstream release that covers
this issue and removing the corresponding custom implementation.

Check these primary sources and the actual released tag.

- https://github.com/Wei-Shaw/sub2api/issues/8609
- https://github.com/Wei-Shaw/sub2api/pull/8622
- https://github.com/openai/codex/issues/42088
- https://github.com/openai/codex/issues/61298
- https://github.com/Wei-Shaw/sub2api/releases

PR #8622 covers specified page snapshots only. A merged PR, issue closure or newer
version number alone does not establish coverage of the three additional names.

Before retiring the patch, verify the official implementation against the three
names, real paired tool outputs, previous-response history and two successive
turns with changing external-context markers. These valid cases must complete
with visible context and correct usage attribution through the public gateway.
A malformed ordinary tool output with no call ID, standalone name/namespace or
historical call context must still produce an explicit error. Official support
for additional legitimate dynamic context names is acceptable. No fixture may
silently lose external context or real tool results.

Follow the existing native-console build and release runbook in Uranus. Preserve
the separately authorized UI integration and external image service. Build the
managed console from the selected upstream release, removing this temporary
normalizer extension and redundant tests when official coverage is verified.
Use private R2 recovery, exact running identity and business verification. Replacing
the managed console with a stock binary would discard the approved UI integration.

If no suitable formal release exists or verification fails, keep this patch and
report the specific upstream status or failing regression. The scheduled review
runs once.
