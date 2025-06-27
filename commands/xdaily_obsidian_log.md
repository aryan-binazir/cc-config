---
description: Format an existing file into a structured Obsidian daily log with specific sections
---

# Format Obsidian Daily Log

Take the specified file and format it into a structured Obsidian daily log using this exact format:

```
---
id: "YYYY-MM-DD"
aliases: []
tags: []
---

# Most Important Task (MIT)

- [ ] [[TASK-ID]] (Priority)

# Tasks Todo

- [ ] Task description [[TICKET-ID]] (Priority)

# Meetings

- [ ] HH:MM Meeting Name (Priority)  

# Other Notes

[>] Additional notes and follow-ups
```

## Formatting Guidelines:

- Use YAML frontmatter with date as ID in YYYY-MM-DD format
- **Preserve all existing [[]] Obsidian links** from the original content
- Use checkbox syntax: [x] for completed items, [>] for ongoing/future items
- Include priority indicators: (High), (Medium), (Low)
- Format ticket/task references as [[TICKET-ID]] for Obsidian linking
- Use HH:MM time format for meeting entries
- Extract the most important task for the MIT section
- Organize remaining tasks under Tasks Todo
- Convert meeting content into time-based entries under Meetings
- Place miscellaneous content in Other Notes section
- Maintain all original links, references, and important context

The output should be clean, Obsidian-compatible, and preserve all [[]] links while organizing content into this structured daily log format.