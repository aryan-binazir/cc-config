<command>
  <metadata>
    <name>xcommit_detailed_v2</name>
    <version>2.0</version>
    <description>Analyze staged changes and create a detailed commit message with comprehensive description</description>
    <complexity>medium</complexity>
    <category>git</category>
  </metadata>
  
  <parameters>
    <parameter name="branch" type="string" required="true">
      <description>Branch name as first argument</description>
      <validation>non_empty_string</validation>
    </parameter>
  </parameters>
  
  <instructions>
    <step id="1" type="check_config">
      <description>Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions</description>
      <action>read_project_config</action>
      <validation>ensure_git_repository</validation>
    </step>
    
    <step id="2" type="git_analysis">
      <description>Run git diff --cached to see all staged changes</description>
      <action>git diff --cached</action>
      <output_variable>staged_changes</output_variable>
      <validation>ensure_staged_changes_exist</validation>
    </step>
    
    <step id="3" type="analysis">
      <description>Analyze the changes and generate a concise but descriptive commit message</description>
      <action>analyze_changes</action>
      <input_variable>staged_changes</input_variable>
      <output_variable>commit_analysis</output_variable>
    </step>
    
    <step id="4" type="message_generation">
      <description>Create commit message in format: "[branch] - [detailed description of changes]"</description>
      <action>generate_commit_message</action>
      <input_variable>commit_analysis</input_variable>
      <output_variable>commit_message</output_variable>
      <template>[{branch}] - {detailed_description}</template>
    </step>
    
    <step id="5" type="comprehensive_description">
      <description>Generate a comprehensive description of the changes in Markdown format</description>
      <action>generate_comprehensive_description</action>
      <input_variable>commit_analysis</input_variable>
      <output_variable>comprehensive_description</output_variable>
      <include>
        <item>Overview of what was changed and why</item>
        <item>Detailed breakdown of modifications by file/component</item>
        <item>Impact and benefits of the changes</item>
        <item>Any technical details worth noting</item>
      </include>
      <exclude>
        <item>Claude co-authorship or other unnecessary details</item>
      </exclude>
    </step>
    
    <step id="6" type="git_operation">
      <description>Run git commit with generated message and comprehensive description</description>
      <action>git commit -m "{commit_message}" -m "{comprehensive_description}"</action>
      <input_variable>commit_message</input_variable>
      <input_variable>comprehensive_description</input_variable>
      <validation>check_commit_success</validation>
    </step>
  </instructions>
  
  <error_handling>
    <error type="no_staged_changes">
      <message>No staged changes found. Use 'git add' to stage files first.</message>
      <action>exit_gracefully</action>
    </error>
    <error type="commit_failed">
      <message>Git commit failed. Check for pre-commit hooks or conflicts.</message>
      <action>exit_gracefully</action>
    </error>
    <error type="invalid_branch_name">
      <message>Branch name parameter is required and cannot be empty</message>
      <action>exit_gracefully</action>
    </error>
  </error_handling>
  
  <output>
    <format>structured</format>
    <template>
Commit created: {commit_message}

Comprehensive Description:
{comprehensive_description}
    </template>
  </output>
  
  <usage>
    <description>Pass branch name as first argument</description>
    <requirements>
      <item>Must be in a git repository</item>
      <item>Must have staged changes</item>
    </requirements>
    <sample_calls>
      <call>xcommit_detailed_v2 feature/new-api</call>
      <call>xcommit_detailed_v2 main</call>
      <call>xcommit_detailed_v2 bugfix/critical-fix</call>
    </sample_calls>
  </usage>
</command>