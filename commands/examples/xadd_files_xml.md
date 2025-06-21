<command>
  <metadata>
    <name>xadd_files</name>
    <version>1.0</version>
    <description>Add all modified and new files to git staging area (git add .)</description>
    <complexity>simple</complexity>
  </metadata>
  
  <parameters>
    <!-- No parameters required for this command -->
  </parameters>
  
  <instructions>
    <step id="1" type="check_config">
      <description>Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions</description>
      <action>read_project_config</action>
    </step>
    
    <step id="2" type="git_operation">
      <description>Add all modified and new files to git staging area</description>
      <action>git add .</action>
      <validation>check_git_status</validation>
    </step>
  </instructions>
  
  <usage>
    <example>Run from any git repository to stage all changes</example>
  </usage>
</command>