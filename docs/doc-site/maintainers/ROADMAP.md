---
title: "Roadmap"
description: "Let's share our plans."
weight: 5
---

## What's next with this project?

{{< mermaid align="center" zoom="true" >}}
timeline
    title Planned releases
    section Q1 2026
    ✅ v0.32.x (March 2026) : Repo carved out of go-swagger
                    : relint
                    : library setup (not env. sensitive)
                    : go1.25+
    section Q2 2026
    ✅ v0.33.x (April 2026) : Reduced exposed interface
                    : type array for parameters
                    : new package layout (internal, layered)
    ✅ v0.34.x (May 2026) : Grammar-based parser
                    : Replace regexp-based parser by lexer+grammar
                    : Fixed many parsing quirks
    ✅ v0.35.0 (June 2026) : Large bug-bashing
                    : Documentation site
                    : Fixes ~200+ go-swagger issues
                    : All validations
                    : Parser diagnostics
    section Q3 2026
    ✅ v0.35.x (July 2026) : Minor features
                    : more tunable knobs, new annotations
                    : Name conflict handling & circular $ref, missing validations, ... 
                    : go doc filter, private comments, inner markdown
    ✅ v0.36.0 (July 2026) : TUI
                    : interactive spec building with TUI tool
                    : polymorphic subtypes discovery
    🔶 v0.36.x (August 2026) : faster code scanner
                    : Optimized incremental type scanner
                    : dedicated CLI, independent from go-swagger
    ⬜ v0.37.0 (September 2026) : decouple from `Spec`
                    : go1.26+
                    : Internal model
                    : More go-swagger backlog fixes & tunable knobs
    section Q4 2026
    🔍 v0.38.x (Oct 2026) : LSP & IDE integration, playground UI (tentative)
    🔍 v0.39.x (Oct-Nov 2026) : OAI v3 support (tentative)
{{< /mermaid >}}
