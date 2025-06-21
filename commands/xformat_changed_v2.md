<command>
  <metadata>
    <name>xformat_changed_v2</name>
    <version>2.0</version>
    <description>Format only Git-changed files with project-configured formatters</description>
    <complexity>complex</complexity>
    <category>formatting</category>
  </metadata>
  
  <parameters>
    <!-- No required parameters, operates on git changes -->
  </parameters>
  
  <configuration>
    <config_sources priority="1">
      <source type="file">package.json</source>
      <scripts>format, prettier, format-code</scripts>
    </config_sources>
    <config_sources priority="2">
      <source type="file">pyproject.toml</source>
      <sections>tool.black, tool.ruff.format</sections>
    </config_sources>
    <config_sources priority="3">
      <source type="file">.prettierrc, .editorconfig</source>
    </config_sources>
    <config_sources priority="4">
      <source type="file">Cargo.toml, go.mod</source>
      <fallback>language_defaults</fallback>
    </config_sources>
  </configuration>
  
  <languages>
    <language name="go" extensions=".go">
      <tool>gofmt</tool>
      <additional>goimports</additional>
      <config_files>go.mod</config_files>
    </language>
    <language name="javascript" extensions=".js,.jsx">
      <tool>prettier</tool>
      <config_files>.prettierrc, .prettierrc.json, .prettierrc.js, prettier.config.js</config_files>
    </language>
    <language name="typescript" extensions=".ts,.tsx">
      <tool>prettier</tool>
      <config_files>.prettierrc, .prettierrc.json, .prettierrc.js, prettier.config.js</config_files>
    </language>
    <language name="python" extensions=".py">
      <tool>pyink</tool>
      <fallback>black</fallback>
      <config_files>pyproject.toml, .black, setup.cfg</config_files>
    </language>
    <language name="rust" extensions=".rs">
      <tool>rustfmt</tool>
      <config_files>rustfmt.toml, .rustfmt.toml</config_files>
    </language>
    <language name="java" extensions=".java">
      <tool>google-java-format</tool>
      <fallback>spotless</fallback>
      <config_files>spotless.gradle, .editorconfig</config_files>
    </language>
  </languages>
  
  <instructions>
    <step id="1" type="check_config">
      <description>Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions</description>
      <action>read_project_config</action>
      <validation>ensure_git_repository</validation>
    </step>
    
    <step id="2" type="git_analysis">
      <description>Get changed files from Git (staged, unstaged, or branch changes)</description>
      <action>get_changed_files</action>
      <output_variable>changed_files</output_variable>
      <validation>ensure_files_exist</validation>
    </step>
    
    <step id="3" type="config_detection">
      <description>Check project config files for formatter preferences and custom commands</description>
      <action>detect_project_formatters</action>
      <config_priority>custom_scripts > project_configs > language_defaults</config_priority>
      <output_variable>formatter_config</output_variable>
    </step>
    
    <step id="4" type="file_grouping">
      <description>Group changed files by language and apply appropriate formatter</description>
      <action>group_by_language</action>
      <input_variable>changed_files</input_variable>
      <output_variable>grouped_files</output_variable>
    </step>
    
    <step id="5" type="conditional_processing">
      <description>Apply formatters respecting project-specific configurations and settings</description>
      <for_each>grouped_files</for_each>
      <conditional>
        <if condition="custom_script_exists">
          <action>run_custom_script</action>
          <parameters>
            <script>{formatter_config.custom_script}</script>
            <files>{language.files}</files>
          </parameters>
        </if>
        <else_if condition="language_supported">
          <action>run_language_formatter</action>
          <parameters>
            <formatter_command>{language.tool}</formatter_command>
            <files>{language.files}</files>
            <config>{language.config_files}</config>
          </parameters>
          <fallback>
            <action>run_fallback_formatter</action>
            <formatter_command>{language.fallback}</formatter_command>
          </fallback>
        </else_if>
        <else>
          <action>log_unsupported_language</action>
          <message>No formatter configured for {language}</message>
        </else>
      </conditional>
    </step>
    
    <step id="6" type="reporting">
      <description>Report formatting results</description>
      <action>generate_format_report</action>
      <input_variable>format_results</input_variable>
    </step>
  </instructions>
  
  <error_handling>
    <error type="no_changed_files">
      <message>No changed files found in git</message>
      <action>exit_gracefully</action>
    </error>
    <error type="formatter_not_found">
      <message>Formatter {formatter_name} not available for {language}</message>
      <action>try_fallback_formatter</action>
    </error>
    <error type="config_file_error">
      <message>Configuration file error: {config_file}</message>
      <action>use_default_settings</action>
    </error>
    <error type="formatting_failed">
      <message>Formatting failed for {language} files: {error_details}</message>
      <action>continue_with_other_languages</action>
    </error>
  </error_handling>
  
  <output>
    <format>structured</format>
    <template>
Formatting Results:
==================

Files processed: {total_files}
Languages detected: {languages_found}
Custom scripts used: {custom_scripts}

{per_language_results}

Summary:
- Successfully formatted: {formatted_count}
- Failed to format: {failed_count}
- Skipped files: {skipped_count}
    </template>
  </output>
  
  <usage>
    <description>Run from project root directory</description>
    <requirements>
      <item>Must be in a git repository</item>
      <item>Appropriate formatters must be installed for detected languages</item>
      <item>Changed files must exist in git</item>
    </requirements>
    <formatter_priority>
      <item>1. Custom scripts in package.json/pyproject.toml</item>
      <item>2. Project config files (.prettierrc, pyproject.toml)</item>
      <item>3. Language defaults (gofmt, rustfmt)</item>
    </formatter_priority>
    <supported_languages>Go, JavaScript/TypeScript, Python, Rust, Java</supported_languages>
  </usage>
</command>