---
name: meeting-notes-cleaner
description: Clean up raw meeting notes into a structured, actionable, Obsidian-compatible summary while preserving meaning, links, and important technical details. Use when the user wants meeting notes cleaned, organized, or reformatted for Obsidian.
---

# Meeting Notes Cleaner

Turn rough meeting notes into a professional, actionable markdown summary.

## Workflow

1. Verify the target notes file exists.
2. Create a timestamped backup before editing.
3. Read the notes and identify attendees, topics, decisions, action items, dates, and existing `[[Obsidian]]` links.
4. Remove filler and redundancy without changing meaning.
5. Organize the content into a consistent markdown structure.
6. Preserve all meaningful technical details and existing wiki links.
7. Add missing `[[links]]` for important people, projects, or concepts when clearly appropriate.
8. Make action items explicit with owners and due dates when present in the notes.

## Output Template

```markdown
# Meeting: [Topic] - [Date]

## Attendees
- [[Person 1]] - Role

## Key Decisions
> Important decision with rationale

## Discussion Points
### Topic 1
- Main points and conclusions

## Action Items
- [ ] **[[Owner]]** - Task description (Due: YYYY-MM-DD)

## Next Steps
- Follow-up meeting, milestones, dependencies
```

## Style

- Use Obsidian-compatible markdown.
- Prefer concise bullets and active voice.
- Highlight action items with bold owners.
- Format decisions as blockquotes.
