# Claude Code XML Command Template

## Standard Schema for XML-based Commands

```xml
<command>
  <!-- METADATA SECTION -->
  <metadata>
    <name>command_name</name>
    <version>1.0</version>
    <description>Brief description of what this command does</description>
    <complexity>simple|medium|complex</complexity>
    <category>git|linting|formatting|docs|analysis</category>
  </metadata>
  
  <!-- PARAMETERS SECTION -->
  <parameters>
    <parameter name="param_name" type="string|number|boolean|array" required="true|false">
      <description>Description of the parameter</description>
      <default>default_value</default>
      <validation>regex_pattern_or_rule</validation>
    </parameter>
  </parameters>
  
  <!-- LANGUAGE CONFIGURATION (for multi-language commands) -->
  <languages>
    <language name="language_name" extensions=".ext1,.ext2">
      <tool>primary_tool_command</tool>
      <fallback>fallback_tool_command</fallback>
      <config_files>config1.json, config2.yaml</config_files>
      <flags>--flag1 --flag2</flags>
    </language>
  </languages>
  
  <!-- CONFIGURATION DETECTION -->
  <configuration>
    <config_sources priority="1|2|3">
      <source type="file|command|env">source_name</source>
      <fallback>fallback_source</fallback>
    </config_sources>
  </configuration>
  
  <!-- MAIN INSTRUCTION FLOW -->
  <instructions>
    <step id="1" type="check_config|git_operation|analysis|processing|reporting">
      <description>Human-readable description of this step</description>
      <action>actual_command_or_operation</action>
      <input_variable>variable_from_previous_step</input_variable>
      <output_variable>variable_for_next_steps</output_variable>
      <validation>validation_rule_or_command</validation>
      
      <!-- For conditional steps -->
      <conditional>
        <if condition="condition_expression">
          <action>action_if_true</action>
        </if>
        <else>
          <action>action_if_false</action>
        </else>
      </conditional>
      
      <!-- For iterative steps -->
      <for_each>variable_containing_list</for_each>
    </step>
  </instructions>
  
  <!-- ERROR HANDLING -->
  <error_handling>
    <error type="error_category">
      <message>User-friendly error message with {placeholders}</message>
      <action>exit_gracefully|retry|skip|fallback</action>
      <fallback_command>alternative_command</fallback_command>
    </error>
  </error_handling>
  
  <!-- USAGE AND EXAMPLES -->
  <usage>
    <description>How to use this command</description>
    <requirements>
      <item>Requirement 1</item>
      <item>Requirement 2</item>
    </requirements>
    <example>Basic usage example</example>
    <sample_calls>
      <call>command arg1 arg2</call>
      <call scenario="advanced">command --flag arg1</call>
    </sample_calls>
  </usage>
  
  <!-- OUTPUT FORMATTING (optional) -->
  <output>
    <format>json|markdown|plain|structured</format>
    <template>Output template with {placeholders}</template>
  </output>
</command>
```

## Usage Guidelines

### When to Use Each Section:

**Always Include:**
- `metadata` - Basic command information
- `instructions` - Core command flow  
- `usage` - Help information

**Include for Complex Commands:**
- `parameters` - For commands accepting arguments
- `languages` - For multi-language tools (linting, formatting, etc.)
- `configuration` - For commands that detect project configs
- `error_handling` - For robust error management
- `output` - For structured output formatting

**Optional Sections:**
- `conditional` - For if/else logic within steps
- `for_each` - For iterating over collections
- `validation` - For input/output validation

### Best Practices:

1. **Use semantic step types**: `check_config`, `git_operation`, `analysis`, `processing`, `reporting`
2. **Clear variable flow**: Use `input_variable` and `output_variable` for data passing
3. **Structured error handling**: Define specific error types with appropriate actions
4. **Validation at key points**: Validate inputs, intermediate results, and final outputs
5. **Language-agnostic design**: Use configuration sections for language-specific behavior

This template provides the foundation for converting your existing commands and creating new ones with improved structure and maintainability.