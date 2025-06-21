<command>
  <metadata>
    <name>xadd_files_v2</name>
    <version>2.0</version>
    <description>Add all modified and new files to git staging area (git add .)</description>
    <complexity>simple</complexity>
    <category>git</category>
  </metadata>
  
  <parameters>
    <!-- No parameters required for this command -->
  </parameters>
  
  <instructions>
    <step id="1" type="check_config">
      <description>Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions</description>
      <action>read_project_config</action>
      <validation>ensure_git_repository</validation>
    </step>
    
    <step id="2" type="git_operation">
      <description>Add all modified and new files to git staging area</description>
      <action>git add .</action>
      <validation>check_git_status_after_add</validation>
    </step>
  </instructions>
  
  <error_handling>
    <error type="not_git_repository">
      <message>Not in a git repository</message>
      <action>exit_gracefully</action>
    </error>
    <error type="no_changes_to_add">
      <message>No changes found to add</message>
      <action>exit_gracefully</action>
    </error>
  </error_handling>
  
  <usage>
    <description>Run from any git repository to stage all changes</description>
    <requirements>
      <item>Must be in a git repository</item>
    </requirements>
    <example>Automatically stages all modified and new files</example>
  </usage>
</command>