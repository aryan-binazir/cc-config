<command>
  <metadata>
    <name>xapi_diff_v2</name>
    <version>2.0</version>
    <description>Show API changes in modified files from Git</description>
    <complexity>complex</complexity>
    <category>analysis</category>
  </metadata>
  
  <parameters>
    <!-- No required parameters, operates on git changes -->
  </parameters>
  
  <languages>
    <language name="javascript" extensions=".js,.jsx">
      <analyzers>functions, classes, exports, modules</analyzers>
      <patterns>
        <function>function\s+(\w+)|const\s+(\w+)\s*=\s*(?:async\s+)?(?:\([^)]*\)|[^=]+)\s*=></function>
        <class>class\s+(\w+)</class>
        <export>export\s+(?:default\s+)?(?:function|class|const|let|var)\s+(\w+)</export>
      </patterns>
    </language>
    <language name="typescript" extensions=".ts,.tsx">
      <analyzers>functions, classes, interfaces, types, exports</analyzers>
      <patterns>
        <function>function\s+(\w+)|const\s+(\w+)\s*:\s*(?:\([^)]*\)\s*=>\s*\w+|\([^)]*\)\s*=></function>
        <class>class\s+(\w+)</class>
        <interface>interface\s+(\w+)</interface>
        <type>type\s+(\w+)\s*=</type>
        <export>export\s+(?:default\s+)?(?:function|class|interface|type|const|let|var)\s+(\w+)</export>
      </patterns>
    </language>
    <language name="python" extensions=".py">
      <analyzers>functions, classes, decorators, imports</analyzers>
      <patterns>
        <function>def\s+(\w+)\s*\(</function>
        <class>class\s+(\w+)\s*[\(:]</class>
        <decorator>@(\w+)</decorator>
        <import>from\s+[\w\.]+\s+import\s+(\w+)|import\s+([\w\.]+)</import>
      </patterns>
    </language>
    <language name="go" extensions=".go">
      <analyzers>functions, structs, interfaces, methods</analyzers>
      <patterns>
        <function>func\s+(\w+)\s*\(</function>
        <struct>type\s+(\w+)\s+struct</struct>
        <interface>type\s+(\w+)\s+interface</interface>
        <method>func\s+\([^)]+\)\s+(\w+)\s*\(</method>
      </patterns>
    </language>
    <language name="java" extensions=".java">
      <analyzers>methods, classes, interfaces, annotations</analyzers>
      <patterns>
        <method>(?:public|private|protected)?\s*(?:static)?\s*[\w<>]+\s+(\w+)\s*\(</method>
        <class>(?:public|private|protected)?\s*class\s+(\w+)</class>
        <interface>(?:public|private|protected)?\s*interface\s+(\w+)</interface>
        <annotation>@(\w+)</annotation>
      </patterns>
    </language>
  </languages>
  
  <api_analysis>
    <change_types>
      <added>new_functions, new_classes, new_endpoints, new_exports</added>
      <modified>changed_signatures, changed_parameters, changed_return_types</modified>
      <removed>deleted_functions, deleted_classes, made_private</removed>
      <breaking>signature_changes, removed_public_apis, type_changes</breaking>
    </change_types>
    <detection_rules>
      <breaking_change>parameter_removal, parameter_type_change, return_type_change, public_to_private</breaking_change>
      <non_breaking>parameter_addition_with_default, new_optional_parameter, additional_return_fields</non_breaking>
    </detection_rules>
  </api_analysis>
  
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
      <description>Filter files that likely contain API definitions</description>
      <action>filter_api_files</action>
      <input_variable>changed_files</input_variable>
      <output_variable>api_files</output_variable>
      <include_patterns>*.js, *.ts, *.py, *.go, *.java, *.cs</include_patterns>
      <exclude_patterns>*.test.*, *.spec.*, *test*, *spec*</exclude_patterns>
    </step>
    
    <step id="4" type="before_after_analysis">
      <description>Analyze API changes in modified files</description>
      <action>analyze_api_changes</action>
      <input_variable>api_files</input_variable>
      <output_variable>api_changes</output_variable>
      <analyze_for>
        <item>Function signatures (parameters, return types)</item>
        <item>Class/interface definitions</item>
        <item>Endpoint routes and methods</item>
        <item>Public method additions/removals</item>
        <item>Breaking changes detection</item>
      </analyze_for>
    </step>
    
    <step id="5" type="change_categorization">
      <description>Categorize changes as added, modified, removed, or breaking</description>
      <action>categorize_changes</action>
      <input_variable>api_changes</input_variable>
      <output_variable>categorized_changes</output_variable>
      <categories>
        <added>new_apis</added>
        <modified>changed_apis</modified>
        <removed>deleted_apis</removed>
        <breaking>breaking_changes</breaking>
      </categories>
    </step>
    
    <step id="6" type="rest_api_analysis">
      <description>Analyze REST API changes if OpenAPI/Swagger files are present</description>
      <action>analyze_rest_api_changes</action>
      <input_variable>api_files</input_variable>
      <output_variable>rest_api_changes</output_variable>
      <conditional>
        <if condition="openapi_files_present">
          <action>parse_openapi_changes</action>
          <analyze>endpoints, methods, schemas, parameters</analyze>
        </if>
      </conditional>
    </step>
    
    <step id="7" type="report_generation">
      <description>Generate structured diff showing API changes</description>
      <action>generate_api_diff_report</action>
      <input_variable>categorized_changes</input_variable>
      <input_variable>rest_api_changes</input_variable>
      <output_variable>api_diff_report</output_variable>
      <format>readable_structured_diff</format>
    </step>
    
    <step id="8" type="export">
      <description>Export results in readable format</description>
      <action>export_api_diff</action>
      <input_variable>api_diff_report</input_variable>
      <formats>console, markdown, json</formats>
    </step>
  </instructions>
  
  <error_handling>
    <error type="no_changed_files">
      <message>No changed files found in git</message>
      <action>exit_gracefully</action>
    </error>
    <error type="no_api_files">
      <message>No API-related files found in changes</message>
      <action>exit_gracefully</action>
    </error>
    <error type="parsing_error">
      <message>Failed to parse {language} file: {file_name}</message>
      <action>skip_file_continue</action>
    </error>
    <error type="analysis_failed">
      <message>API analysis failed for {file_name}: {error_details}</message>
      <action>skip_file_continue</action>
    </error>
  </error_handling>
  
  <output>
    <format>structured</format>
    <template>
API Changes Analysis:
====================

Files analyzed: {analyzed_files_count}
Languages detected: {languages_found}

Added APIs:
{added_apis_list}

Modified APIs:
{modified_apis_list}

Removed APIs:
{removed_apis_list}

⚠️ Breaking Changes:
{breaking_changes_list}

REST API Changes:
{rest_api_changes}

Summary:
- New APIs: {added_count}
- Modified APIs: {modified_count}  
- Removed APIs: {removed_count}
- Breaking changes: {breaking_count}
- Non-breaking changes: {non_breaking_count}
    </template>
  </output>
  
  <usage>
    <description>Run from project root directory</description>
    <requirements>
      <item>Must be in a git repository</item>
      <item>Changed files must exist in git</item>
      <item>Files should contain API definitions</item>
    </requirements>
    <supported_languages>JavaScript/TypeScript, Python, Go, Java, REST APIs</supported_languages>
    <analysis_scope>
      <item>Functions, classes, interfaces</item>
      <item>Endpoint routes and methods</item>
      <item>Public method additions/removals</item>
      <item>Breaking vs non-breaking changes</item>
    </analysis_scope>
  </usage>
</command>