---
description: Clean up and organize meeting notes while preserving meaning and formatting for Obsidian
version: "1.1"
---

# Clean Up Meeting Notes

Transform raw meeting notes into a professional, actionable Obsidian-compatible summary.

## Process:

1. **Input validation:**
   - Verify the specified file exists and is readable
   - Check if file contains meeting content (not empty)
   - Create backup copy with timestamp: `filename.backup.YYYY-MM-DD-HHMMSS.md`

2. **Content analysis:**
   - Identify meeting structure: agenda items, discussions, decisions
   - Extract key participants, dates, and action items
   - Preserve all existing [[]] Obsidian links and references
   - Note any missing critical information (attendees, date, etc.)

3. **Clean and organize:**
   - Remove filler words, redundant phrases, and unnecessary repetition
   - Consolidate similar points into single, clear statements
   - Organize information into logical sections
   - Preserve all key decisions, action items, and important details
   - **Preserve all existing [[]] Obsidian links** from the original content
   - Maintain the original meaning while improving clarity and readability

4. **Generate output:**
   - Apply structured format (see template below)
   - Create new [[]] links for important people, projects, or concepts mentioned
   - Ensure all action items have clear ownership and deadlines
   - Add summary section if meeting was lengthy or complex

## Output Format:
- Meeting title and date at the top (use # header)
- Attendees list (if available) (use ## header)
- Key discussion points organized by topic (use ## headers for topics)
- Clear action items with assigned owners and deadlines
- Important decisions highlighted with **bold** text or > blockquotes
- Next steps clearly outlined (use ## header)
- Preserve all [[]] Obsidian links and references

## Style Guidelines:
- Output in clean, Obsidian-compatible Markdown
- Use bullet points for easy scanning
- Keep sentences concise but complete
- Use active voice where possible
- **Preserve all existing [[]] links** and create new ones for important references
- Remove verbal fillers and tangential discussions
- Preserve technical terms and specific details
- Maintain chronological flow when relevant
- Use proper Markdown headers (##, ###) for section organization
- Highlight action items with **bold** text
- Use > blockquotes for important decisions or key takeaways

## Output Template:
```markdown
# Meeting: [Topic] - [Date]

## Attendees
- [[Person 1]] - Role
- [[Person 2]] - Role

## Key Decisions
> Important decision 1 with rationale
> Important decision 2 with rationale

## Discussion Points
### Topic 1
- Main points discussed
- Concerns raised
- Conclusions reached

### Topic 2
- [Similar structure]

## Action Items
- [ ] **[[Owner]]** - Task description (Due: YYYY-MM-DD)
- [ ] **[[Owner]]** - Task description (Due: YYYY-MM-DD)

## Next Steps
- Follow-up meeting: [Date/Time]
- Key milestones: [List]
- Dependencies: [List]

## Notes
- Additional context or references
- [[Links to related documents]]
```

## Guidelines:
- **Preserve context**: Don't sacrifice important details for brevity
- **Enhance clarity**: Remove verbal fillers but keep technical specifics  
- **Action-oriented**: Ensure every action item has clear ownership and deadlines
- **Link-rich**: Create [[]] links for people, projects, documents mentioned
- **Scannable**: Use consistent formatting for quick reference

## Safety Features:
- Creates timestamped backup before modifying original
- Preserves all existing [[]] links and references
- Maintains chronological flow when relevant

Usage: Specify the meeting notes file path as an argument
