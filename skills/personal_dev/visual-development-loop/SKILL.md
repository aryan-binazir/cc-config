---
name: visual-development-loop
description: Drive browser UI development from observed visual evidence using an existing local or deployed application, isolated browser automation, responsive viewports, constrained cold loads, DOM measurements, and deterministic component fixtures when live state is unreliable. Use whenever the user asks for screenshot-driven development, browser-driven UI iteration, visual reproduction before editing, responsive-layout fixes, loading-flash or layout-shift investigation, or before-and-after Chrome verification.
---

# Visual Development Loop

Develop browser interfaces from reproduced evidence rather than assumptions. Combine screenshots with DOM, network, console, accessibility, and layout measurements so a visually plausible result does not hide a functional defect.

## Boundaries

- Preserve the surrounding task's authorization boundary. A diagnosis or review request stays read-only; an implementation request permits scoped product changes.
- Reuse an already-running application when available. Do not start or restart development servers, builds, containers, or supporting services unless the user explicitly asks.
- Keep visual tooling ephemeral. Put temporary browser profiles, screenshots, traces, rendered fixtures, and helper scripts outside the repository and remove them when they are no longer needed.
- Do not create reports, committed screenshots, permanent fixtures, new project scripts, dependencies, or other deliverables unless the user explicitly requests them.
- Inspect the repository and runtime before choosing tools. Do not assume a framework, package manager, port, route structure, browser, authentication provider, API layout, or deployment topology.
- Do not bypass authentication, CORS, TLS, browser security, or application authorization to make a reproduction work. Diagnose the boundary instead.
- Protect session material. Never print or persist cookies, bearer tokens, OAuth parameters, storage contents, or URLs containing credentials. Use an isolated browser profile and let the user complete interactive authentication when necessary.

## Workflow

### 1. Define the evidence matrix

Translate the reported behavior into the smallest set of observable cases before editing:

- exact page, component, interaction, and application state;
- representative desktop and mobile widths, using project-defined targets when available;
- initial load, in-app navigation, or both;
- warm and cold cache when startup behavior matters;
- constrained network or CPU only when latency exposes the defect;
- relevant theme, content length, empty/error/loading state, and authentication state.

Use `1440px` and `390px` only as reasonable defaults when the project has no stated viewport targets.

### 2. Verify the runtime and ownership

- Check whether the expected application URL is already healthy before opening a browser.
- Identify the component, styles, state owner, API boundary, and existing tests responsible for the visible behavior.
- Inspect project instructions and design-language sources before judging the intended result.
- If the required application is not running and starting it was not authorized, stop and report the concrete missing prerequisite.

### 3. Establish an isolated browser session

Prefer the browser automation already available in the environment. Chrome DevTools Protocol, Playwright, or an equivalent tool is acceptable when it can control the real browser behavior under investigation.

- Use a disposable or dedicated automation profile so normal browser windows remain untouched.
- Reuse an authenticated automation profile when safe; otherwise ask the user to authenticate in the isolated window.
- Sanitize browser-target and navigation output so query parameters or session material cannot leak into logs.
- Confirm the browser reached the intended origin and page rather than silently redirecting to sign-in, an error page, or another deployment.

### 4. Capture the baseline

Reproduce the defect before changing production code. For startup or loading defects, disable cache and sample several points across the load rather than relying on one final screenshot.

Collect only the evidence relevant to the issue, such as:

- viewport and document dimensions;
- document-level and container-level overflow;
- visible text and loading or error states;
- bounding boxes for clipped, wrapped, or displaced elements;
- cumulative layout shift or equivalent layout movement;
- console errors and failed requests;
- focus order, accessible names, disabled state, and keyboard behavior when interaction is involved.

Treat screenshots as one observation, not the verdict. A screenshot cannot prove that the correct route loaded, the API succeeded, the layout did not shift, or an interaction works.

### 5. Use deterministic fixtures only when needed

When live data cannot reliably produce a required visual state, render the real component through its normal public interface with controlled representative data.

- Keep the component and its actual styles under test; avoid hand-recreating its markup.
- Cover long content, empty data, loading, error, and boundary values only when they matter to the reported defect.
- Keep fixtures temporary unless the surrounding task independently justifies durable test coverage.
- Prefer the live application for routing, authentication, loading, network, and integration behavior. A static fixture cannot validate those paths.

### 6. Implement in red-green slices

For each confirmed defect:

1. Add or update a failing test at the real public seam when the behavior is testable outside a browser.
2. Make the smallest product change that resolves the reproduced case.
3. Run the focused test immediately.
4. Repeat for the next independent behavior.

Do not extract production code solely to make it testable, and do not add a browser-test framework for a single visual check unless the repository already uses it or the user approves the new dependency.

### 7. Repeat the same browser checks

Re-run the original evidence matrix after the change. Keep viewport, application state, cache, and throttling conditions equivalent so before-and-after observations are comparable.

Verify both appearance and behavior:

- no unintended document overflow, clipping, wrapping, or overlap;
- loading content appears promptly and resolves without a large jump;
- intended internal scroll regions remain usable and visibly discoverable;
- desktop behavior did not regress while fixing mobile behavior, or vice versa;
- copy and visible values match the actual state;
- controls remain reachable, legible, and operable;
- console and network state contain no new failures.

Compare against the project's existing design language rather than inventing a redesign.

### 8. Run repository gates and clean up

- Run the project's focused tests, then its required typecheck, lint, formatting, and broader test or build checks in proportion to the change.
- Report exactly what passed, failed, or was not exercised.
- Remove temporary profiles, screenshots, traces, HTML fixtures, and helper scripts created for the investigation.
- Leave no visual-development artifacts in the repository or final diff unless the user explicitly requested durable tooling or evidence.

## Completion

Finish with a concise status covering the behavior verified, viewport and runtime conditions exercised, repository checks run, and any remaining limitation. Do not generate a separate visual report or artifact bundle.
