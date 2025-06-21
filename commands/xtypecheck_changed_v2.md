<command>
  <metadata>
    <name>xtypecheck_changed_v2</name>
    <version>2.0</version>
    <description>Type check only Git-changed files with language-specific tools</description>
    <complexity>complex</complexity>
    <category>typechecking</category>
  </metadata>
  
  <parameters>
    <!-- No required parameters, operates on git changes -->
  </parameters>
  
  <configuration>
    <config_sources priority="1">
      <source type="file">package.json</source>
      <scripts>typecheck, type-check, tsc</scripts>
    </config_sources>
    <config_sources priority="2">
      <source type="file">pyproject.toml</source>
      <sections>tool.mypy, tool.pyright</sections>
    </config_sources>
    <config_sources priority="3">
      <source type="file">tsconfig.json, mypy.ini</source>
    </config_sources>
  </configuration>
  
  <languages>
    <language name="typescript" extensions=".ts,.tsx">
      <tool>tsc --noEmit</tool>
      <config_files>tsconfig.json, tsconfig.*.json</config_files>
      <flags>--incremental</flags>
    </language>
    <language name="python" extensions=".py">
      <tool>mypy</tool>
      <fallback>pyright</fallback>
      <additional>pyre</additional>
      <config_files>pyproject.toml, mypy.ini, .mypy.ini, setup.cfg</config_files>
    </language>
    <language name="go" extensions=".go">
      <tool>go vet</tool>
      <additional>staticcheck</additional>
      <config_files>go.mod</config_files>
    </language>
    <language name="java" extensions=".java">
      <tool>javac</tool>
      <additional>Error Prone</additional>
      <config_files>build.gradle, pom.xml</config_files>
    </language>
    <language name="csharp" extensions=".cs">
      <tool>dotnet build</tool>
      <config_files>*.csproj, *.sln</config_files>
      <flags>--no-restore</flags>
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
    
    <step id="3" type="file_filtering">
      <description>Filter files by language and run appropriate type checker</description>
      <action>filter_by_language</action>
      <input_variable>changed_files</input_variable>
      <output_variable>typed_files</output_variable>
    </step>
    
    <step id="4" type="config_detection">
      <description>Check project config files for custom type checking commands</description>
      <action>detect_typecheck_config</action>
      <config_priority>custom_scripts > project_configs > language_defaults</config_priority>
      <output_variable>typecheck_config</output_variable>
    </step>
    
    <step id="5" type="conditional_processing">
      <description>Report type errors only for changed files and their dependencies</description>
      <for_each>typed_files</for_each>
      <conditional>
        <if condition="custom_script_exists">
          <action>run_custom_typecheck</action>
          <parameters>
            <script>{typecheck_config.custom_script}</script>
            <files>{language.files}</files>
          </parameters>
        </if>
        <else_if condition="language_supported">
          <action>run_language_typecheck</action>
          <parameters>
            <typecheck_command>{language.tool}</typecheck_command>
            <files>{language.files}</files>
            <config>{language.config_files}</config>
            <flags>{language.flags}</flags>
          </parameters>
          <fallback>
            <action>run_fallback_typecheck</action>
            <typecheck_command>{language.fallback}</typecheck_command>
          </fallback>
        </else_if>
        <else>
          <action>log_unsupported_language</action>
          <message>No type checker configured for {language}</message>
        </else>
      </conditional>
    </step>
    
    <step id="6" type="dependency_analysis">
      <description>Include type errors from dependencies of changed files</description>
      <action>analyze_dependencies</action>
      <input_variable>typecheck_results</input_variable>
      <output_variable>complete_results</output_variable>
    </step>
    
    <step id="7" type="reporting">
      <description>Generate type checking report focused on changed files</description>
      <action>generate_typecheck_report</action>
      <input_variable>complete_results</input_variable>
      <focus>changed_files_and_dependencies</focus>
    </step>
  </instructions>
  
  <error_handling>
    <error type="no_changed_files">
      <message>No changed files found in git</message>
      <action>exit_gracefully</action>
    </error>
    <error type="no_typed_files">
      <message>No files requiring type checking found in changes</message>
      <action>exit_gracefully</action>
    </error>
    <error type="typechecker_not_found">
      <message>Type checker {typechecker_name} not available for {language}</message>
      <action>try_fallback_typechecker</action>
    </error>
    <error type="config_file_error">
      <message>Type checking configuration error: {config_file}</message>
      <action>use_default_settings</action>
    </error>
    <error type="typecheck_execution_failed">
      <message>Type checking failed for {language} files: {error_details}</message>
      <action>continue_with_other_languages</action>
    </error>
  </error_handling>
  
  <output>
    <format>structured</format>
    <template>
Type Checking Results:
=====================

Files processed: {total_files}
Languages detected: {languages_found}
Custom scripts used: {custom_scripts}

{per_language_results}

Summary:
- Files with type errors: {error_files_count}
- Total type errors: {total_errors}
- Clean files: {clean_files_count}
- Skipped files: {skipped_count}

Focus: Changed files and their dependencies
    </template>
  </output>
  
  <usage>
    <description>Run from project root directory</description>
    <requirements>
      <item>Must be in a git repository</item>
      <item>Appropriate type checkers must be installed for detected languages</item>
      <item>Changed files must exist in git</item>
    </requirements>
    <supported_typecheckers>
      <item>TypeScript: tsc, @typescript-eslint</item>
      <item>Python: mypy, pyright, pyre</item>
      <item>Go: go vet, staticcheck</item>
      <item>Java: javac, Error Prone</item>
      <item>C#: dotnet build</item>
    </supported_typecheckers>
  </usage>
</command>