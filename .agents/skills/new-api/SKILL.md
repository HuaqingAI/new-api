```markdown
# new-api Development Patterns

> Auto-generated skill from repository analysis

## Overview
This skill teaches the core development patterns and conventions used in the `new-api` Python codebase. It covers file naming, import/export styles, commit message conventions, and testing patterns. The repository follows a clean and consistent structure, making it easy to maintain and extend.

## Coding Conventions

### File Naming
- Use **kebab-case** for all file names.
  - Example: `user-service.py`, `data-handler.py`

### Import Style
- Use **relative imports** within the package.
  - Example:
    ```python
    from .utils import parse_data
    ```

### Export Style
- Use **named exports** (explicitly define what is exported).
  - Example:
    ```python
    def fetch_data():
        pass

    __all__ = ['fetch_data']
    ```

### Commit Messages
- Follow **Conventional Commits** with prefixes like `chore`, `docs`.
  - Example: `chore: update dependencies`
  - Example: `docs: add API usage examples`

## Workflows

### Code Update
**Trigger:** When making any code changes or improvements  
**Command:** `/update-code`

1. Create or update Python files using kebab-case naming.
2. Use relative imports for internal modules.
3. Explicitly define exports with `__all__`.
4. Write clear, conventional commit messages (e.g., `chore: refactor data handler`).

### Documentation Update
**Trigger:** When updating or adding documentation  
**Command:** `/update-docs`

1. Edit or add documentation files.
2. Use the `docs:` prefix in commit messages.
3. Ensure examples follow the codebase conventions.

### Testing
**Trigger:** When adding or updating tests  
**Command:** `/run-tests`

1. Create test files using the `*.test.*` pattern (e.g., `user-service.test.py`).
2. Write tests using your preferred framework (framework not specified).
3. Run tests to verify code correctness.

## Testing Patterns

- Test files are named with the pattern `*.test.*` (e.g., `api-handler.test.py`).
- The testing framework is not specified; use your team's standard.
- Place tests alongside the code or in a dedicated tests directory.

## Commands
| Command        | Purpose                                  |
|----------------|------------------------------------------|
| /update-code   | Apply code changes following conventions  |
| /update-docs   | Update or add documentation              |
| /run-tests     | Run all test files matching `*.test.*`    |
```
