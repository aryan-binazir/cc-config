<command>
  <metadata>
    <name>xcommit</name>
    <version>1.0</version>
    <description>Analyze staged changes and create a detailed commit message</description>
    <complexity>medium</complexity>
  </metadata>
  
  <parameters>
    <parameter name="branch" type="string" required="true">
      <description>Branch name as first argument</description>
      <default>current_branch</default>
    </parameter>
  </parameters>
  
  <instructions>
    <step id="1" type="check_config">
      <description>Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions</description>
      <action>read_project_config</action>
    </step>
    
    <step id="2" type="git_analysis">
      <description>Run git diff --cached to see all staged changes</description>
      <action>git diff --cached</action>
      <validation>ensure_staged_changes_exist</validation>
    </step>
    
    <step id="3" type="analysis">
      <description>Analyze the changes and generate a concise but descriptive commit message</description>
      <action>analyze_changes</action>
      <output_variable>commit_analysis</output_variable>
    </step>
    
    <step id="4" type="message_generation">
      <description>Create commit message in format: "[branch] - [detailed description of changes]"</description>
      <action>generate_commit_message</action>
      <template>[{branch}] - {detailed_description}</template>
      <output_variable>commit_message</output_variable>
    </step>
    
    <step id="5" type="git_operation">
      <description>Run git commit with generated message</description>
      <action>git commit -m "{commit_message}"</action>
      <validation>check_commit_success</validation>
    </step>
  </instructions>
  
  <usage>
    <example>Pass branch name as first argument</example>
    <sample_call>xcommit feature/new-api</sample_call>
  </usage>
</command>