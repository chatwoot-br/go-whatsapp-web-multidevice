# Documentation Guide

A comprehensive guide for maintaining and updating the documentation in this project.

## Table of Contents

- [Documentation Philosophy](#documentation-philosophy)
- [Documentation Structure](#documentation-structure)
- [When to Update Documentation](#when-to-update-documentation)
- [Where to Add New Documentation](#where-to-add-new-documentation)
- [Documentation Standards](#documentation-standards)
- [Common Scenarios](#common-scenarios)
- [Review Process](#review-process)
- [Tools and Workflows](#tools-and-workflows)
- [Maintenance Checklist](#maintenance-checklist)

## Documentation Philosophy

This project follows the **[Divio Documentation System](https://documentation.divio.com/)**, which organizes documentation into four categories:

```
┌─────────────────┬─────────────────┐
│   TUTORIALS     │     HOW-TO      │
│  (learning)     │   (problem)     │
│                 │                 │
│ getting-started/│    guides/      │
└─────────────────┴─────────────────┘
┌─────────────────┬─────────────────┐
│   REFERENCE     │  EXPLANATION    │
│ (information)   │ (understanding) │
│                 │                 │
│  reference/     │   developer/    │
└─────────────────┴─────────────────┘
```

### Key Principles

1. **Single Source of Truth** - No duplicate information
2. **Clear Organization** - Easy to find what you need
3. **Comprehensive Coverage** - All features documented
4. **Practical Examples** - Real, working code examples
5. **Cross-Referenced** - Links to related documentation
6. **Up-to-Date** - Documentation updated with code changes

## Documentation Structure

```
docs/
├── README.md                    # Main navigation hub
│
├── getting-started/             # 📚 TUTORIALS - Learning-oriented
│   ├── quick-start.md          # Get running in 5 minutes
│   ├── installation.md         # Complete installation guide
│   ├── first-message.md        # Send your first message
│   └── configuration-basics.md # Essential configuration
│
├── guides/                      # 🎯 HOW-TO - Problem-oriented
│   ├── deployment/             # Deployment methods
│   ├── webhooks/               # Webhook integration
│   ├── mcp-integration.md      # MCP server setup
│   ├── admin-api.md            # Multi-instance management
│   └── media-handling.md       # Media processing
│
├── reference/                   # 📖 REFERENCE - Information-oriented
│   ├── api/                    # API specifications
│   ├── webhooks/               # Webhook schemas
│   ├── configuration.md        # All config options
│   ├── phone-number-format.md  # JID format reference
│   └── troubleshooting.md      # Common issues
│
├── developer/                   # 🔧 EXPLANATION - Understanding-oriented
│   ├── architecture.md         # System design
│   ├── contributing.md         # How to contribute
│   ├── testing.md              # Testing guide
│   ├── release-process.md      # Creating releases
│   ├── documentation-guide.md  # This file
│   └── adr/                    # Architecture decisions
│
├── operations/                  # ⚙️ OPERATIONS - Running in production
│   ├── monitoring.md           # Metrics and logging
│   ├── performance-tuning.md   # Optimization
│   ├── security-best-practices.md
│   └── audio-optimization.md
│
└── postmortems/                # 📝 LESSONS LEARNED
    ├── 001-profile-picture-panic.md
    ├── 002-multidevice-encryption.md
    └── lessons-learned.md
```

## When to Update Documentation

Documentation should be updated in these situations:

### ✅ Always Update Documentation For

1. **New Features**
   - API endpoints
   - Configuration options
   - Command-line flags
   - New modes or capabilities

2. **Breaking Changes**
   - API changes
   - Configuration changes
   - Behavior changes
   - Migration requirements

3. **Bug Fixes** (if user-facing)
   - Known issues resolved
   - Workarounds no longer needed
   - Behavior corrections

4. **Configuration Changes**
   - New environment variables
   - Changed defaults
   - Deprecated options

5. **Deployment Changes**
   - New deployment methods
   - Updated requirements
   - Infrastructure changes

### ⚠️ Consider Updating Documentation For

1. **Internal Refactoring** (if affects understanding)
   - Architecture changes
   - Major code reorganization
   - New design patterns

2. **Performance Improvements**
   - New optimization options
   - Changed resource requirements
   - Scaling characteristics

3. **Security Updates**
   - New security features
   - Security best practices
   - Vulnerability fixes

### ❌ Don't Document

1. Internal implementation details (unless architectural)
2. Temporary workarounds (use code comments instead)
3. Work-in-progress features (wait until stable)

## Where to Add New Documentation

Use this decision tree to determine where new documentation belongs:

### Step 1: What type of documentation?

```
Is it a tutorial for beginners?
├─ YES → getting-started/
└─ NO
    │
    Is it a how-to guide for a specific task?
    ├─ YES → guides/
    └─ NO
        │
        Is it reference material (specs, schemas, configs)?
        ├─ YES → reference/
        └─ NO
            │
            Is it explanation of architecture/concepts?
            ├─ YES → developer/
            └─ NO
                │
                Is it operations/production guidance?
                ├─ YES → operations/
                └─ NO → Ask in PR review
```

### Step 2: Specific Guidelines by Type

#### getting-started/

**Add here if:**
- Teaching beginners how to get started
- Step-by-step tutorial format
- Assumes no prior knowledge
- Goal is to achieve first success

**Examples:**
- "Quick Start Guide"
- "Your First WhatsApp Message"
- "Basic Configuration"

**Don't add:**
- Advanced topics
- Reference material
- Troubleshooting (use reference/)

#### guides/

**Add here if:**
- Solving a specific problem
- Goal-oriented instructions
- Assumes some familiarity
- Practical, actionable steps

**Subdirectories:**
- `deployment/` - Deployment methods
- `webhooks/` - Webhook integration
- Root level - Other how-to guides

**Examples:**
- "How to Deploy with Docker"
- "How to Set Up Webhooks"
- "How to Handle Media Files"

**Don't add:**
- Beginner tutorials (use getting-started/)
- API specs (use reference/)

#### reference/

**Add here if:**
- Technical specifications
- Complete reference material
- Lookup/search usage
- Information-dense

**Subdirectories:**
- `api/` - API specifications
- `webhooks/` - Webhook schemas
- Root level - Other reference docs

**Examples:**
- OpenAPI specifications
- Configuration reference
- Webhook payload schemas
- Phone number format

**Don't add:**
- Tutorials (use getting-started/)
- How-to guides (use guides/)

#### developer/

**Add here if:**
- Explaining architecture
- Contributing guidelines
- Understanding system design
- Background/context

**Examples:**
- Architecture overview
- Contributing guide
- Testing strategy
- Architecture Decision Records (ADRs)

**Don't add:**
- User-facing documentation
- API reference (use reference/)

#### operations/

**Add here if:**
- Production operations
- Monitoring and observability
- Performance optimization
- Security in production

**Examples:**
- Monitoring guide
- Performance tuning
- Security best practices

**Don't add:**
- Deployment how-to (use guides/)
- Development setup (use developer/)

#### postmortems/

**Add here if:**
- Significant production incident
- Valuable lessons learned
- Root cause analysis
- Prevention strategies

**Examples:**
- Critical bug postmortems
- Service outages
- Major incidents

**Don't add:**
- Minor bugs (use GitHub Issues)
- In-progress issues

## Documentation Standards

### File Naming

```bash
# Use kebab-case for file names
✅ quick-start.md
✅ mcp-integration.md
✅ phone-number-format.md

# NOT CamelCase or snake_case
❌ QuickStart.md
❌ quick_start.md
```

### Document Structure

Every document should have:

```markdown
# Title (H1 - only one per document)

Brief introduction (2-3 sentences) explaining what this document covers.

## Table of Contents (for docs >200 lines)

- [Section 1](#section-1)
- [Section 2](#section-2)

## Section 1

Content...

### Subsection 1.1

More content...

## Related Documentation

Links to related docs...

---

**Last Updated**: 2025-12-05
**Version Compatibility**: v7.10.0+
```

### Writing Style

1. **Be Clear and Concise**
   ```markdown
   ✅ "Run the server with `./whatsapp rest`"
   ❌ "You can start the server by executing the binary with the rest argument"
   ```

2. **Use Active Voice**
   ```markdown
   ✅ "Create a new file called `.env`"
   ❌ "A new file called `.env` should be created"
   ```

3. **Provide Examples**
   ```markdown
   ✅
   Set the webhook URL:
   ```bash
   WHATSAPP_WEBHOOK=https://example.com/webhook
   ```

   ❌ "Configure the webhook URL in environment variables"
   ```

4. **Use Code Blocks with Language**
   ```markdown
   ✅ ```bash
      curl http://localhost:3000/app/login
      ```

   ❌ ```
      curl http://localhost:3000/app/login
      ```
   ```

### Markdown Formatting

#### Headers

```markdown
# H1 - Document Title (only one)
## H2 - Main Sections
### H3 - Subsections
#### H4 - Sub-subsections (use sparingly)
```

#### Code Examples

```markdown
# Inline code
Use `backticks` for inline code, commands, and file names.

# Code blocks
Use fenced code blocks with language:
```bash
./whatsapp rest --port 3000
```

```json
{
  "phone": "6281234567890",
  "message": "Hello"
}
```
```

#### Lists

```markdown
# Unordered lists
- First item
- Second item
  - Nested item (2 spaces)
  - Another nested item

# Ordered lists
1. Step one
2. Step two
3. Step three

# Task lists (GitHub)
- [ ] Uncompleted task
- [x] Completed task
```

#### Links

```markdown
# Internal links (preferred)
See [Configuration Guide](../reference/configuration.md)

# External links
See [Divio Documentation System](https://documentation.divio.com/)

# Anchor links
See [Configuration](#configuration) in this document
```

#### Tables

```markdown
| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Value 1  | Value 2  | Value 3  |
| Value 4  | Value 5  | Value 6  |
```

#### Admonitions

```markdown
> **Note**: Important information for users
> **Warning**: Critical information, potential issues
> **Tip**: Helpful suggestions
> **Important**: Must-read information
```

### Cross-Referencing

Always cross-reference related documentation:

```markdown
## Related Documentation

- [API Reference](../reference/api/openapi.md) - Complete API documentation
- [Webhook Guide](../guides/webhooks/setup.md) - Setting up webhooks
- [Configuration](../reference/configuration.md) - All configuration options
```

## Common Scenarios

### Scenario 1: Adding a New API Endpoint

**Files to Update:**

1. **`docs/reference/api/openapi.yaml`** (REQUIRED)
   - Add endpoint specification
   - Include request/response schemas
   - Add examples

2. **`docs/reference/api/openapi.md`** (RECOMMENDED)
   - Add endpoint to relevant section
   - Include usage examples

3. **README.md** (if major feature)
   - Add to feature list
   - Add to API endpoints table

4. **`docs/guides/`** (if complex)
   - Create how-to guide if needed

**Example PR Checklist:**
```markdown
- [ ] Updated openapi.yaml with new endpoint
- [ ] Added examples to openapi.md
- [ ] Updated README.md feature list
- [ ] Added how-to guide (if needed)
- [ ] Tested examples work
```

### Scenario 2: Adding a New Configuration Option

**Files to Update:**

1. **`docs/reference/configuration.md`** (REQUIRED)
   - Add to appropriate category
   - Include: variable name, default, description, example

2. **`src/.env.example`** (REQUIRED)
   - Add example configuration

3. **`docs/getting-started/configuration-basics.md`** (if essential)
   - Add if it's a commonly used option

4. **CLAUDE.md** (if developer-relevant)
   - Update configuration section

5. **README.md** (if major feature)
   - Add to configuration table

**Example:**

In `docs/reference/configuration.md`:
```markdown
| Variable | Default | Description | Example |
|----------|---------|-------------|---------|
| `NEW_FEATURE_ENABLED` | `false` | Enable new feature XYZ | `NEW_FEATURE_ENABLED=true` |
```

### Scenario 3: Fixing a Bug

**Files to Update:**

1. **`docs/reference/troubleshooting.md`** (if user-facing)
   - Remove workaround if one was documented
   - Update error description if needed

2. **`docs/postmortems/`** (if critical bug)
   - Create postmortem document
   - Document root cause and fix

3. **CHANGELOG.md** (REQUIRED)
   - Document the bug fix

### Scenario 4: Adding a New Feature

**Files to Create/Update:**

1. **Create feature guide** in `docs/guides/`
   ```
   docs/guides/new-feature.md
   ```

2. **Update API documentation** (if has API)
   ```
   docs/reference/api/openapi.yaml
   docs/reference/api/openapi.md
   ```

3. **Update getting-started** (if essential)
   ```
   docs/getting-started/quick-start.md
   ```

4. **Update main docs**
   ```
   docs/README.md (add to relevant section)
   README.md (add to features list)
   CHANGELOG.md (document new feature)
   ```

### Scenario 5: Deprecating a Feature

**Files to Update:**

1. **Mark as deprecated** in documentation
   ```markdown
   > **Deprecated**: This feature is deprecated as of v7.10.0.
   > Use [new-feature](new-feature.md) instead.
   ```

2. **Update all references**
   - Reference docs
   - Guides
   - Examples

3. **Add migration guide** (if complex)
   ```
   docs/guides/migrating-from-old-to-new.md
   ```

4. **Update CHANGELOG**
   - Document deprecation
   - Provide timeline for removal

### Scenario 6: Major Architecture Change

**Files to Update:**

1. **Create ADR** (Architecture Decision Record)
   ```
   docs/developer/adr/NNNN-description.md
   ```

2. **Update architecture.md**
   ```
   docs/developer/architecture.md
   ```

3. **Update relevant guides**
   - Deployment guides if needed
   - Developer documentation

4. **Create migration guide** (if breaking)
   ```
   docs/guides/migration-vX-to-vY.md
   ```

## Review Process

### Self-Review Checklist

Before submitting a PR with documentation changes:

- [ ] **Accuracy**: All information is correct and tested
- [ ] **Completeness**: All required documentation updated
- [ ] **Examples**: Code examples work and are tested
- [ ] **Links**: All links work (internal and external)
- [ ] **Formatting**: Proper markdown formatting
- [ ] **Spelling**: No typos or grammar errors
- [ ] **Cross-References**: Related docs are linked
- [ ] **Consistency**: Follows existing style and structure
- [ ] **Table of Contents**: Updated if document structure changed
- [ ] **Version Info**: Updated version compatibility notes

### Documentation Review Guidelines

When reviewing documentation PRs:

1. **Check Accuracy**
   - Test all code examples
   - Verify technical accuracy
   - Ensure version compatibility

2. **Check Placement**
   - Is it in the right section?
   - Should it be split differently?
   - Is there redundancy?

3. **Check Quality**
   - Clear and understandable?
   - Good examples?
   - Proper formatting?

4. **Check Completeness**
   - All related docs updated?
   - Navigation updated?
   - Cross-references added?

## Tools and Workflows

### Local Documentation Preview

```bash
# Preview markdown locally
# Option 1: Use VS Code markdown preview (Cmd/Ctrl+Shift+V)

# Option 2: Use grip (GitHub-flavored markdown)
pip install grip
grip docs/README.md

# Option 3: Use mkdocs (if configured)
mkdocs serve
```

### Validate Markdown

```bash
# Install markdownlint
npm install -g markdownlint-cli

# Lint all markdown files
markdownlint docs/**/*.md

# Fix auto-fixable issues
markdownlint --fix docs/**/*.md
```

### Check Links

```bash
# Install markdown-link-check
npm install -g markdown-link-check

# Check all links in a file
markdown-link-check docs/README.md

# Check all markdown files
find docs -name "*.md" -exec markdown-link-check {} \;
```

### Find Documentation TODOs

```bash
# Find TODO items in documentation
grep -r "TODO" docs/
grep -r "FIXME" docs/
grep -r "STUB" docs/
```

### Generate Table of Contents

```bash
# Using markdown-toc
npm install -g markdown-toc

# Generate TOC
markdown-toc -i docs/README.md
```

## Maintenance Checklist

### Weekly Maintenance

- [ ] Review recent PRs for missing documentation
- [ ] Check for broken links
- [ ] Review and close documentation issues

### Monthly Maintenance

- [ ] Update getting-started guides for current version
- [ ] Review and update troubleshooting guide
- [ ] Check for outdated information
- [ ] Update version compatibility notes
- [ ] Review stub files and complete them

### Per-Release Maintenance

- [ ] Update all version numbers in docs
- [ ] Update CHANGELOG.md
- [ ] Review breaking changes documentation
- [ ] Update migration guides if needed
- [ ] Update API specs with new endpoints
- [ ] Update configuration reference
- [ ] Review and update README.md feature list

### Quarterly Review

- [ ] Review entire documentation structure
- [ ] Identify gaps in coverage
- [ ] Remove outdated documentation
- [ ] Consolidate duplicate information
- [ ] Update examples to current best practices
- [ ] Review and update stub files
- [ ] Check documentation metrics (if available)

## Documentation Metrics

Track these metrics to improve documentation quality:

1. **Coverage**
   - % of API endpoints documented
   - % of configuration options documented
   - % of code with examples

2. **Quality**
   - Documentation issues opened
   - Documentation PRs merged
   - User feedback on docs

3. **Freshness**
   - Last updated dates
   - Outdated sections
   - Broken links count

4. **Usage** (if analytics available)
   - Most visited pages
   - Search queries
   - Time on page

## Best Practices

### DO ✅

1. **Update docs with code changes**
   - Documentation in same PR as code
   - Don't wait until later

2. **Test all examples**
   - Copy-paste and run
   - Verify they work

3. **Use consistent terminology**
   - Refer to style guide
   - Same terms throughout

4. **Keep it up-to-date**
   - Regular maintenance
   - Remove outdated info

5. **Write for your audience**
   - Beginners in getting-started/
   - Experienced users in reference/

6. **Include screenshots** (when helpful)
   - UI documentation
   - Complex processes
   - Store in `docs/images/`

### DON'T ❌

1. **Don't duplicate information**
   - Link to single source of truth
   - Consolidate when possible

2. **Don't write huge documents**
   - Split into focused docs
   - Max ~800 lines per file

3. **Don't forget cross-references**
   - Always link related docs
   - Help users discover content

4. **Don't use emojis excessively**
   - Use sparingly for emphasis
   - Maintain professional tone

5. **Don't write code in documentation**
   - Put code in codebase
   - Document the interface

6. **Don't assume knowledge**
   - Define terms
   - Link to explanations

## Getting Help

### Questions About Documentation

1. **Structure questions**: Ask in PR or issue
2. **Style questions**: Refer to this guide
3. **Technical questions**: Ask code reviewers

### Improving This Guide

This guide itself can be improved! If you find:

- Missing scenarios
- Unclear instructions
- Better practices
- Outdated information

Please open a PR to update `docs/developer/documentation-guide.md`.

## Resources

### External Resources

- [Divio Documentation System](https://documentation.divio.com/) - Our documentation philosophy
- [Google Developer Documentation Style Guide](https://developers.google.com/style)
- [Write the Docs](https://www.writethedocs.org/) - Documentation best practices
- [Markdown Guide](https://www.markdownguide.org/) - Markdown syntax reference

### Internal Resources

- [Contributing Guide](contributing.md) - How to contribute to this project
- [Architecture Overview](architecture.md) - Understanding the system
- [Release Process](release-process.md) - Creating releases

---

**Last Updated**: 2025-12-05
**Maintained By**: Development Team
**Version**: 1.1.0
