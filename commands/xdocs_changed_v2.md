<command>
  <metadata>
    <name>xdocs_changed_v2</name>
    <version>2.0</version>
    <description>Update documentation affected by code changes in Git</description>
    <complexity>complex</complexity>
    <category>docs</category>
  </metadata>
  
  <parameters>
    <!-- No required parameters, operates on git changes -->
  </parameters>
  
  <documentation_types>
    <doc_type name="api_docs" patterns="**/api/**/*.md, **/docs/api/**">
      <triggers>function_changes, endpoint_changes, interface_changes</triggers>
      <generators>OpenAPI, JSDoc, docstrings</generators>
    </doc_type>
    <doc_type name="readme" patterns="README.md, **/README.md">
      <triggers>feature_changes, config_changes, installation_changes</triggers>
      <sections>installation, usage, configuration, examples</sections>
    </doc_type>
    <doc_type name="config_docs" patterns="**/docs/config/**/*.md, CONFIG.md">
      <triggers>config_file_changes, env_var_changes, settings_changes</triggers>
      <generators>config_schema, env_documentation</generators>
    </doc_type>
    <doc_type name="architecture" patterns="**/docs/architecture/**/*.md, ARCHITECTURE.md">
      <triggers>structural_changes, module_changes, dependency_changes</triggers>
      <generators>architecture_diagrams, dependency_graphs</generators>
    </doc_type>
    <doc_type name="changelog" patterns="CHANGELOG.md, **/CHANGELOG.md">
      <triggers>any_changes</triggers>
      <format>keep_a_changelog</format>
    </doc_type>
  </documentation_types>
  
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
    
    <step id="3" type="change_analysis">
      <description>Identify documentation that needs updates based on code changes</description>
      <action>analyze_code_changes</action>
      <input_variable>changed_files</input_variable>
      <output_variable>change_analysis</output_variable>
      <analyze_for>
        <item>API documentation for changed endpoints/functions</item>
        <item>README sections for modified features</item>
        <item>Config documentation for changed settings</item>
        <item>Architecture docs for structural changes</item>
      </analyze_for>
    </step>
    
    <step id="4" type="documentation_mapping">
      <description>Map code changes to documentation files that need updates</description>
      <action>map_changes_to_docs</action>
      <input_variable>change_analysis</input_variable>
      <output_variable>docs_to_update</output_variable>
      <mapping_rules>
        <rule>function_changes -> api_docs</rule>
        <rule>config_changes -> config_docs + readme</rule>
        <rule>feature_changes -> readme + changelog</rule>
        <rule>structural_changes -> architecture + readme</rule>
      </mapping_rules>
    </step>
    
    <step id="5" type="conditional_processing">
      <description>Generate or update relevant documentation files</description>
      <for_each>docs_to_update</for_each>
      <conditional>
        <if condition="doc_type == 'api_docs'">
          <action>generate_api_documentation</action>
          <parameters>
            <generator>{doc_type.generators}</generator>
            <changed_functions>{change_analysis.functions}</changed_functions>
            <output_format>markdown</output_format>
          </parameters>
        </if>
        <else_if condition="doc_type == 'readme'">
          <action>update_readme_sections</action>
          <parameters>
            <sections>{doc_type.sections}</sections>
            <changes>{change_analysis.features}</changes>
          </parameters>
        </else_if>
        <else_if condition="doc_type == 'config_docs'">
          <action>update_config_documentation</action>
          <parameters>
            <config_changes>{change_analysis.config}</config_changes>
            <schema_generator>{doc_type.generators}</schema_generator>
          </parameters>
        </else_if>
        <else_if condition="doc_type == 'architecture'">
          <action>update_architecture_docs</action>
          <parameters>
            <structural_changes>{change_analysis.structure}</structural_changes>
            <diagram_generator>{doc_type.generators}</diagram_generator>
          </parameters>
        </else_if>
        <else_if condition="doc_type == 'changelog'">
          <action>update_changelog</action>
          <parameters>
            <format>{doc_type.format}</format>
            <changes>{change_analysis.all}</changes>
          </parameters>
        </else_if>
      </conditional>
    </step>
    
    <step id="6" type="validation">
      <description>Validate generated documentation for accuracy and completeness</description>
      <action>validate_documentation</action>
      <input_variable>updated_docs</input_variable>
      <checks>
        <check>links_valid</check>
        <check>examples_work</check>
        <check>schema_valid</check>
        <check>format_consistent</check>
      </checks>
    </step>
    
    <step id="7" type="reporting">
      <description>Suggest documentation updates for manual review</description>
      <action>generate_documentation_report</action>
      <input_variable>updated_docs</input_variable>
      <include_manual_review>true</include_manual_review>
    </step>
  </instructions>
  
  <error_handling>
    <error type="no_changed_files">
      <message>No changed files found in git</message>
      <action>exit_gracefully</action>
    </error>
    <error type="no_documentation_impact">
      <message>Code changes do not affect any documentation</message>
      <action>exit_gracefully</action>
    </error>
    <error type="documentation_generator_failed">
      <message>Failed to generate {doc_type} documentation: {error_details}</message>
      <action>create_manual_task</action>
    </error>
    <error type="validation_failed">
      <message>Documentation validation failed: {validation_errors}</message>
      <action>flag_for_manual_review</action>
    </error>
  </error_handling>
  
  <output>
    <format>structured</format>
    <template>
Documentation Update Results:
=============================

Code changes analyzed: {changed_files_count}
Documentation files affected: {docs_affected_count}

Generated/Updated:
{updated_docs_list}

Manual Review Required:
{manual_review_list}

Validation Results:
- Valid documentation: {valid_count}
- Issues found: {issues_count}
- Manual tasks created: {manual_tasks_count}

Supported Documentation Types:
- API docs (OpenAPI, JSDoc, docstrings)
- README sections
- Configuration guides  
- Architecture diagrams
- Changelog entries
    </template>
  </output>
  
  <usage>
    <description>Run from project root directory</description>
    <requirements>
      <item>Must be in a git repository</item>
      <item>Changed files must exist in git</item>
      <item>Documentation generators must be available if needed</item>
    </requirements>
    <workflow>
      <item>Analyzes code changes for documentation impact</item>
      <item>Maps changes to relevant documentation types</item>
      <item>Generates or updates documentation automatically where possible</item>
      <item>Creates manual review tasks for complex changes</item>
    </workflow>
  </usage>
</command>