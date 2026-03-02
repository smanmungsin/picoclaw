# Instruction for AI implementation
1. Implementation must be read-to-use implement.
2. After implement the project building "Build Go project" execution is a must.
3. If there's any error(s) fix it before proceeding to the next step.
4. Implementation must be in Go best practice.
5. Function implementation block must not be added in the other function block, the Function implementation must be separated and being on the same level as other functions.
6. Function implementation must not be on the top of the file (line 1), prefer to put on the bottom of the file instead.

# Hot-Reloading Implementation
Modules and skills now support hot-reloading using fsnotify. The SkillsLoader watches skill directories and reloads skills dynamically on file changes, without restarting the agent.

# Dependency update
1. The package block must be on the first line of the file.
2. After the package block there must be following by import block.
3. The package and import block must not be added in other lines other than 1. and 2.
4. When update the package and import block please update on the existing block only.