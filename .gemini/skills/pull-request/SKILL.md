# GCSFuse Pull Request Creation Skill

This skill provides a highly automated, step-by-step workflow for Antigravity agents to construct, format, push, and create pull requests for their changes in the GCSFuse repository. Executing this runbook ensures that commits and pull requests strictly follow the repository's standardized templates, author identity alignments, and style checking procedures before PR creation.

---

## 1. Context and Formatting Standards

GCSFuse enforces a structured PR style and title system. Agents must align their branch work with the repository's standard [PR Template](../../../.github/pull_request_template.md) to facilitate clean, review-ready pull requests.

### Pull Request Title Standard
The pull request title and git commit message **MUST** strictly follow the Conventional Commits specification:
```
type(scope): subject
```
* **Types list**:
  * `feat`: A new feature
  * `fix`: A bug fix
  * `docs`: Documentation only changes
  * `style`: Structural style updates (white-space, formatting, etc.)
  * `refactor`: A code modification that neither fixes a bug nor adds a feature
  * `perf`: Performance improvements
  * `test`: Correcting or creating missing tests
  * `build`: Build systems or dependency updates (example scope: `deps`)
  * `ci`: Verification runner scripts and presubmits (example scope: `workflows`)
  * `chore`: Other maintenance shifts that don't modify src or test structures
  * `revert`: Reverting a past merge or commit
* **Scope**: A bracketed descriptor highlighting the specific package or component modified (e.g., `skills`, `internal/cache`, `metrics`, `tools`).
* **Subject**: A clear, concise present-tense statement of the changes made (e.g., `add PR creation skill for GCSFuse`).

---

## 2. Compiling the PR Fields

Agents must synthesize a structured PR body based on the original task requirements and development results, mapping them onto the [pull_request_template.md](../../../.github/pull_request_template.md) sections:

> [!IMPORTANT]
> **Instructional Block Omission Rule**:
> The [PR Template](../../../.github/pull_request_template.md) contains an initial introductory/instructional block (lines 1 to 20) detailing conventional commit rules and types. This block is strictly for reference.
> When compiling and generating the pull request body payload, the agent **MUST REMOVE / OMIT this instructional block** completely. The final PR description body must strictly start with the `### Description` header down to the end of the file.

### Section A: Description
* **Guideline**: Summarize *what* was updated and *why*.
* Detail functional logic changes, additions, structural moves, and documentation additions.
* Clearly state architectural transitions or optimizations applied.

### Section B: Link to Issue
* **Guideline**: Reference the bug tracking index, such as internal bug targets (`b/514754560`) or external GitHub Issue URLs.
* Example: `Fixes b/514754560` or `Closes #123`.

### Section C: Testing Details
* **Guideline**: Report test validation outcomes to ensure functional coverage.
* Divide into:
  1. **Manual**: Detail manual tests ran. E.g., manual compilation verification or verification run commands.
  2. **Unit tests**: Report outcome of unit tests (e.g., package test runs or checkpoint suite coverage).
  3. **Integration tests**: Detail E2E or system suite validations run.

### Section D: Backward Incompatibility
* **Guideline**: Inspect modifications to identify any backwards-incompatible changes.
* Analyze signature changes on exported APIs, removal of support parameters, or config skews. If none, state: `N/A`.

---

## 3. Step-by-Step Execution Workflow

### Step 1: Compilation and Layout Pre-Check
Before creating a PR, you **MUST** ensure the branch is fully compliant with repository verification targets.

1. Invoke the **Build and Style Verification Skill** by executing the single native target:
   ```bash
   make build
   ```
2. Ensure the build finishes with exit code `0`. If not, resolve the compilation or lint warnings immediately before proceeding.

### Step 2: Commit Auto-Format & Skill Changes
Because build verifiers run layout formatters in-place, stage all automatic layout alignment differences.

1. Stash or add untracked/modified updates to the branch:
   ```bash
   git add .
   ```
2. Verify staged items match expected additions:
   ```bash
   git status
   ```
3. Commit local staged components using a commit message that maps exactly to the PR Title format compiled in Section 1:
   ```bash
   git commit -m "type(scope): subject"
   ```

### Step 3: Establish Remote Upstream & Push
Push the local branch to your fork or branch tree upstream:
```bash
# Example syntax: git push -u origin <branch_name>
git push -u origin $(git branch --show-current)
```

### Step 4: Raise PR via GitHub CLI
Utilize the authenticated GitHub CLI (`gh`) to trigger unified PR creation:

1. Draft the PR body by populating the template values compiled in Section 2 into a clean markdown structure.
2. Trigger the PR creation in **Draft** status:
   ```bash
   # Syntax: gh pr create --title "<title>" --body "<body>" --draft
   gh pr create --title "type(scope): subject" --body-file - --draft <<EOF
   ### Description
   [Synthesized PR Description]
   
   ### Link to the issue in case of a bug fix.
   [Bug tracker reference]
   
   ### Testing details
   1. Manual - [Manual validation statement]
   2. Unit tests - [Unit validations statement]
   3. Integration tests - [E2E test suite validation statement]
   
   ### Any backward incompatible change? If so, please explain.
   [Incompatibility statement]
   EOF
   ```
   * > [!IMPORTANT]
   * > **Draft PR Mode Enforcement**: Always pass the `--draft` flag to the creation tool. Publishing in draft mode allows the developer to review diff lines, inspect layout annotations, and verify final pull details inside their browser before officially launching review gates.

---

## 4. Fallback Execution

If the GitHub CLI is unauthenticated or missing scopes:
1. Log a warning in the execution session describing the CLI authentication limit.
2. Construct and print a pre-filled direct web link to create a pull request via the browser:
   ```
   https://github.com/GoogleCloudPlatform/gcsfuse/pull/new/<branch_name>?title=<urlencoded_title>&body=<urlencoded_body>
   ```
3. Output the generated PR body text (omitting the top instructions block as per the Omission Rule) directly in markdown so the user can easily copy and paste it into the browser window if requested.
