<command>
  <metadata>
    <name>xupdate_docs_v2</name>
    <version>2.0</version>
    <description>Evaluate the current project and update documentation</description>
    <complexity>simple</complexity>
    <category>docs</category>
  </metadata>
  
  <parameters>
    <!-- No parameters required for this command -->
  </parameters>
  
  <instructions>
    <step id="1" type="check_config">
      <description>Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions</description>
      <action>read_project_config</action>
      <validation>ensure_project_directory</validation>
    </step>
    
    <step id="2" type="project_evaluation">
      <description>Evaluate the current project</description>
      <action>analyze_project_structure</action>
      <output_variable>project_analysis</output_variable>
      <analyze_for>
        <item>Project structure and organization</item>
        <item>Main functionality and features</item>
        <item>Dependencies and technologies used</item>
        <item>Configuration and setup requirements</item>
      </analyze_for>
    </step>
    
    <step id="3" type="documentation_evaluation">
      <description>Evaluate the current documentation</description>
      <action>analyze_existing_documentation</action>
      <output_variable>docs_analysis</output_variable>
      <check_files>README.md, CLAUDE.md, docs/**, *.md</check_files>
      <evaluate_for>
        <item>Completeness and accuracy</item>
        <item>Missing sections or outdated information</item>
        <item>Documentation structure and organization</item>
      </evaluate_for>
    </step>
    
    <step id="4" type="claude_md_update">
      <description>Update CLAUDE.md with any new functionality</description>
      <action>update_claude_md</action>
      <input_variable>project_analysis</input_variable>
      <input_variable>docs_analysis</input_variable>
      <update_sections>
        <item>Project overview and purpose</item>
        <item>New features and capabilities</item>
        <item>Setup and configuration instructions</item>
        <item>Usage examples and workflows</item>
      </update_sections>
      <validation>ensure_claude_md_accuracy</validation>
    </step>
    
    <step id="5" type="general_docs_update">
      <description>Update other documentation as needed</description>
      <action>update_additional_documentation</action>
      <input_variable>project_analysis</input_variable>
      <input_variable>docs_analysis</input_variable>
      <target_files>README.md, docs/**, other *.md files</target_files>
      <update_types>
        <item>Installation instructions</item>
        <item>Usage examples</item>
        <item>API documentation</item>
        <item>Configuration guides</item>
      </update_types>
    </step>
  </instructions>
  
  <error_handling>
    <error type="not_in_project">
      <message>Not in a valid project directory</message>
      <action>exit_gracefully</action>
    </error>
    <error type="claude_md_not_found">
      <message>CLAUDE.md not found in project root</message>
      <action>create_claude_md</action>
    </error>
    <error type="documentation_update_failed">
      <message>Failed to update documentation: {error_details}</message>
      <action>report_error_continue</action>
    </error>
  </error_handling>
  
  <output>
    <format>structured</format>
    <template>
Documentation Update Results:
============================

Project Analysis:
- Structure: {project_structure}
- Technologies: {technologies_found}
- Features: {main_features}

Documentation Updates:
- CLAUDE.md: {claude_md_status}
- README.md: {readme_status}
- Other docs: {other_docs_count} files updated

Summary:
- Files evaluated: {files_evaluated}
- Files updated: {files_updated}
- New sections added: {new_sections}
- Issues found: {issues_count}
    </template>
  </output>
  
  <usage>
    <description>Run from project root directory</description>
    <requirements>
      <item>Must be in a valid project directory</item>
    </requirements>
    <workflow>
      <item>Analyzes current project structure and functionality</item>
      <item>Evaluates existing documentation for completeness</item>
      <item>Updates CLAUDE.md with new functionality</item>
      <item>Updates other documentation files as needed</item>
    </workflow>
  </usage>
</command>