---
name: personal-agenda
description: Use this skill whenever the user asks about their agenda, calendars, schedule, availability, events, reminders, todos, tasks, due/overdue items, daily planning, changing dates, moving/rescheduling items, recurring reminders, or Compass/Notion-linked goals. Trigger aggressively for planning language like "today", "tomorrow", "this week", "what do I have", "remind me", "change the date", "move this", or "personal agenda".
---

# Personal Agenda

Manage the user's calendars and Apple Reminders with their routing rules. Be concise. Return the result without implementation commentary.

For ordinary agenda requests, do not inspect project files, source repos, `AGENTS.md`, `_scratch`, or local context docs. The relevant runtime context is the current local date/time and private agenda settings. Only inspect repo files when maintaining this skill.

## Fast Path

For read-only local agenda requests, use this sequence:

1. Resolve the current local date/time and timezone.
2. Read `~/.config/codex/personal-agenda/config.json`.
3. Use the configured agenda reader from that file.
4. Read today's calendar events, today's reminders, and overdue reminders.
5. Apply the filters and formatting rules below.

## Safety And Writes

Read freely when the user asks about schedule, calendar, reminders, todos, due items, or planning.

Create, update, or delete items when the instruction is explicit and the target is clear. Ask before writes when:

- Multiple matching calendar events or reminders exist.
- A recurring event or reminder could mean one instance or the whole series.
- A write needs private config that is missing.
- The requested change is broad, destructive, or visible in a shared calendar/list.

For recurring calendar changes, ask whether the change applies to this event only or future events/the series.

## Private Config

Look for optional private config at:

`~/.config/codex/personal-agenda/config.json`

Use it for private values such as:

- `default_calendar`
- private calendar aliases
- excluded passive calendars
- private list aliases the public skill should not name

If the config is missing and a write requires one of these values, ask the user for the missing value. After the user answers, offer to create or update the config. Do not create or update the config silently.

Do not hardcode private calendar names, calendar account emails, private Notion page names, addresses, family member names, or personal routines in this public skill.

## Configured Agenda Reader

Use `local_eventkit_reader.path` from `~/.config/codex/personal-agenda/config.json`. If it is missing, ask the user to configure the agenda reader path.

Use narrow reads with the configured reader:

```bash
"$READER" --action "$EVENT_READ_ACTION" --startDate "$START" --endDate "$END"
"$READER" --action "$REMINDER_READ_ACTION" --dueWithin today --showCompleted false
"$READER" --action "$REMINDER_READ_ACTION" --dueWithin overdue --showCompleted false
```

Do not use broad AppleScript/JXA loops over every calendar, list, and reminder. They can hang on macOS automation.

Priority mapping for Apple Reminders is non-obvious:

- `0`: none
- `1`: high
- `5`: medium
- `9`: low

## Reminder List IDs

Use these configured list IDs where possible. Do not include the private family/shared list's real name in public output or skill text.

- `default_tasks_list_id`: `8C874624-CC09-47E6-BBEE-45312518A49C`
- `autopilot_list_id`: `5FA182DD-E3EF-4817-9367-A06F831C569B`
- `goals_list_id`: `D254108D-70CE-49B2-AA0A-4861040877ED`
- `shopping_list_id`: `8564FA53-B013-4C2B-A924-E2AB783FC542`
- `family_shared_list_id`: `15BAFB8A-6ABD-4A5A-9A15-49995046EBAF`
- `travel_list_id`: `96555CDA-6BD2-49E7-88B2-2B87F0005F7F`

If the active write path only accepts list names, read reminder lists first and match the configured ID to the current list name. If matching fails, ask.

## Reminder Routing

If the user names a specific list, respect it.

Otherwise route reminders as follows:

- Compass/goal-linked reminders go to `goals_list_id`, recurring or not.
- Compass family/shared reminders go to `family_shared_list_id`.
- Recurring non-goal reminders go to `autopilot_list_id`.
- One-off non-goal reminders go to `default_tasks_list_id`.
- Shopping/grocery reminders go to `shopping_list_id` when obvious.
- Travel/trip reminders go to `travel_list_id` when obvious.

Only use `filterList` when the user names a specific list. For general reminder queries, omit `filterList` so results come from all lists.

When creating a reminder, the reminder title must be explicit. If the user gives a date, time, priority, or alarm but omits the reminder text, ask for the title before writing. Do not invent generic titles like "Reminder" or "Alarm".

Create one reminder per item when the user gives a list of items.

When marking a reminder complete with the configured EventKit CLI, use `--action update --id "$ID" --isCompleted true --completionDate "$NOW"`; the raw CLI expects `isCompleted`, not `completed`. Verify with `--action read-by-id --id "$ID"` or a narrow read with `--showCompleted false`, and only report success after `isCompleted` is true or the item disappears from active results.

Tags are only for Compass links. Do not invent or apply non-Compass tags unless the user explicitly names them.

When the user asks to change a reminder's date, update `startDate`, `dueDate`, and any absolute-date alarms to the new value unless they explicitly say otherwise. Relative-offset alarms shift automatically; absolute-date alarms do not. Never update only one of these fields for a date change.

Do not create a duplicate calendar event for a timed reminder. Timed reminders already appear on the calendar.

## Compass / Notion Bridge

Compass is a Notion goal system. Do not hardcode Compass child page names in this public skill.

When Compass routing or goal context is needed:

1. Search Notion for the page titled `compass`.
2. Fetch the `compass` page.
3. Under its `north stars` section, treat direct linked child pages as possible Compass goal pages.
4. Ignore related spaces unless the user explicitly asks for them.
5. Ask before linking a reminder to a Compass page/tag.
6. Fetch the selected Compass child page.
7. Read the tag from the top of the page. The expected pattern is a top line like `tag: #some_tag`.
8. Add that tag to the Apple Reminder.
9. Route the reminder according to the Reminder Routing section.

Do not fetch Compass child page contents for ordinary reminder work unless needed to read the tag, answer a goal-context question, or make an explicit edit.

## Calendar Reads

Resolve relative dates using the current local date/time before querying.

For daily overviews:

- Read today's calendar events broadly across calendars.
- Read reminders due today.
- Read overdue reminders.
- Also treat timed reminders due earlier today as overdue, because an overdue-only filter may miss them.
- Exclude passive calendars such as Birthdays, Holidays, and similar observance calendars unless the user explicitly asks.
- Include all-day events only when they are real commitments.

Do not worry about conflicts by default. If a conflict is obvious, mention it briefly if useful, but do not suggest fixes unless the user asks.

## Calendar Writes

For new calendar events, use the configured `default_calendar` from private config when available. If missing, ask before creating the event.

For reads, broad calendar access is fine. For writes, be conservative because the wrong target calendar can be visible to other people.

## Maintenance

Do not create evals for this skill by default.

When the user says the behavior was wrong or asks "what could you have done better?":

1. Identify the missing or incorrect rule.
2. Propose the exact update to this skill.
3. Ask before editing the skill file.
4. After approval, patch `~/repos/cc-config/skills/personal-agenda/SKILL.md`.
