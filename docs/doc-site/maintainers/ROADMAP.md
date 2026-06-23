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
    ✅ v0.35.x (June 2026) : Large bug-bashing
                    : Documentation site
                    : Fixes ~200+ go-swagger issues
                    : All validations
                    : Parser diagnostics
                    : Name conflict handling & circular $ref, missing validations, ... 
    🔶 v0.36.x (July 2026) : CLI & TUI / faster code scanner
                    : Optimized incremental type scanner
                    : more tunable knobs, new annotations
                    : go doc filter, private comments, inner markdown
    ⬜ v0.37.x (August 2026) : playground UI / decouple from `Spec`
                    : Internal model
                    : Discriminator / polymorphism discovery
                    : More go-swagger backlog fixes & tunable knobs
    section Q3 2026
    🔍 v0.38.x (Sept 2026) : LSP & IDE integration
                    : go1.26+
    section Q4 2026
    🔍 v0.39.x (Oct-Nov 2026) : OAI v3 support (tentative)
{{< /mermaid >}}
