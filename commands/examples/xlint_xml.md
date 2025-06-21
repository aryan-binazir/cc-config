<command>
  <metadata>
    <name>xlint</name>
    <version>1.0</version>
    <description>Lint only files changed in Git with appropriate language-specific linters and auto-fix</description>
    <complexity>complex</complexity>
  </metadata>
  
  <parameters>
    <!-- No required parameters, operates on git changes -->
  </parameters>
  
  <languages>
    <language name="javascript" extensions=".js,.jsx">
      <linter>eslint --fix</linter>
      <config_files>eslintrc, .eslintrc.js, .eslintrc.json</config_files>
    </language>
    <language name="typescript" extensions=".ts,.tsx">
      <linter>eslint --fix</linter>
      <config_files>eslintrc, .eslintrc.js, .eslintrc.json, tsconfig.json</config_files>
    </language>
    <language name="python" extensions=".py">
      <linter>ruff check --fix</linter>
      <config_files>pyproject.toml, ruff.toml, .ruff.toml</config_files>
    </language>
    <language name="rust" extensions=".rs">
      <linter>cargo clippy --fix --allow-dirty</linter>
      <config_files>Cargo.toml</config_files>
    </language>
    <language name="go" extensions=".go">
      <linter>golangci-lint run --fix</linter>
      <config_files>go.mod, .golangci.yml</config_files>
    </language>
    <language name="java" extensions=".java">
      <linter>checkstyle</linter>
      <fallback>spotbugs</fallback>
      <config_files>checkstyle.xml, spotbugs.xml</config_files>
    </language>
  </languages>
  
  <instructions>
    <step id="1" type="check_config">
      <description>Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions</description>
      <action>read_project_config</action>
    </step>
    
    <step id="2" type="git_analysis">
      <description>Get changed files from Git (staged, unstaged, or branch changes)</description>
      <action>get_changed_files</action>
      <output_variable>changed_files</output_variable>
      <validation>ensure_files_exist</validation>
    </step>
    
    <step id="3" type="file_grouping">
      <description>Group files by language extension</description>
      <action>group_by_language</action>
      <input_variable>changed_files</input_variable>
      <output_variable>grouped_files</output_variable>
    </step>
    
    <step id="4" type="conditional_processing">
      <description>Run appropriate linter with --fix flag on changed files only</description>
      <for_each>grouped_files</for_each>
      <conditional>
        <if condition="language_supported">
          <action>run_linter_for_language</action>
          <parameters>
            <linter_command>{language.linter}</linter_command>
            <files>{language.files}</files>
            <config>{language.config_files}</config>
          </parameters>
        </if>
        <else>
          <action>log_unsupported_language</action>
        </else>
      </conditional>
    </step>
    
    <step id="5" type="reporting">
      <description>Report any remaining issues</description>
      <action>generate_lint_report</action>
      <include_fixed>true</include_fixed>
      <include_remaining>true</include_remaining>
    </step>
  </instructions>
  
  <error_handling>
    <error type="no_changed_files">
      <message>No changed files found in git</message>
      <action>exit_gracefully</action>
    </error>
    <error type="linter_not_found">
      <message>Linter not available for {language}</message>
      <action>skip_language</action>
    </error>
    <error type="config_error">
      <message>Configuration file error for {language}</message>
      <action>use_default_config</action>
    </error>
  </error_handling>
  
  <usage>
    <description>Run from project root directory</description>
    <requirements>
      <item>Must be in a git repository</item>
      <item>Appropriate linters must be installed</item>
    </requirements>
    <example>Works automatically on any supported language files</example>
  </usage>
</command>