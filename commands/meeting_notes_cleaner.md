---
description: Clean up and organize meeting notes for Obsidian
version: "2.0"
---

# Clean Up Meeting Notes

Transform raw meeting notes into a professional, actionable Obsidian-compatible summary.

## Process:

1. **Input validation:**
   - Verify file exists and create backup: `filename.backup.YYYY-MM-DD-HHMMSS.md`

2. **Content analysis:**
   - Identify structure: agenda items, discussions, decisions
   - Extract participants, dates, action items
   - Preserve all existing [[]] Obsidian links

3. **Clean and organize:**
   - Remove filler words and redundancy
   - Consolidate similar points
   - Preserve key decisions and action items
   - Maintain original meaning while improving clarity

4. **Generate output:**
   - Apply structured format (template below)
   - Create [[]] links for important people, projects, concepts
   - Ensure action items have ownership and deadlines

## Output Template:
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

## Style:
- Obsidian-compatible Markdown with ## headers
- Bullet points, concise sentences, active voice
- Preserve [[]] links and technical details
- Highlight action items with **bold**, decisions with > blockquotes

Usage: Specify the meeting notes file path as an argument
