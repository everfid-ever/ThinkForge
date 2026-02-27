package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ReactConfig ReAct 执行器配置
type ReactConfig struct {
	MaxIterations int                 // 最大推理轮数，默认 5
	Model         model.BaseChatModel // LLM 实例
	Registry      *ToolRegistry       // 工具注册表
}

// ReactResult ReAct 执行结果
type ReactResult struct {
	Answer         string
	References     []*schema.Document
	ReasoningSteps []ReasoningStep
}

// ReactExecutor ReAct 循环执行器
type ReactExecutor struct {
	config *ReactConfig
}

// NewReactExecutor 创建 ReAct 执行器
func NewReactExecutor(config *ReactConfig) *ReactExecutor {
	if config.MaxIterations <= 0 {
		config.MaxIterations = 5
	}
	return &ReactExecutor{config: config}
}

// Run 执行 ReAct 循环
func (e *ReactExecutor) Run(ctx context.Context, intent *RAGIntent, question string, knowledgeName string, topK int, score float64) (*ReactResult, error) {
	toolDescs := e.config.Registry.BuildToolDescriptions()

	systemPrompt := fmt.Sprintf(`You are a professional AI assistant that uses the ReAct (Reasoning + Acting) framework.

Available tools:
%s

Instructions:
1. Analyze the question step by step using Thought → Action → Observation cycles
2. Use tools to retrieve relevant information when needed
3. When you have enough information to answer, output "Final Answer: <your answer>"

Format:
Thought: <your reasoning about what to do>
Action: <tool_name>
Action Input: <JSON input for the tool>

OR when ready to answer:
Thought: <your final reasoning>
Final Answer: <your complete answer>

Current question: %s`, toolDescs, question)

	messages := []*schema.Message{
		{Role: schema.System, Content: systemPrompt},
		{Role: schema.User, Content: question},
	}

	var steps []ReasoningStep
	var allRefs []*schema.Document
	stepNum := 0

	for i := 0; i < e.config.MaxIterations; i++ {
		resp, err := e.config.Model.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("react: llm generate failed at iteration %d: %w", i, err)
		}

		content := resp.Content
		now := time.Now().Format(time.RFC3339)

		// Check for Final Answer
		if idx := strings.Index(content, "Final Answer:"); idx >= 0 {
			// Extract thought if present before final answer
			if thoughtContent := extractThought(content[:idx]); thoughtContent != "" {
				stepNum++
				steps = append(steps, ReasoningStep{
					Step:      stepNum,
					Type:      "thought",
					Content:   thoughtContent,
					Timestamp: now,
				})
			}
			answer := strings.TrimSpace(content[idx+len("Final Answer:"):])
			stepNum++
			steps = append(steps, ReasoningStep{
				Step:      stepNum,
				Type:      "final_answer",
				Content:   answer,
				Timestamp: now,
			})
			return &ReactResult{
				Answer:         answer,
				References:     allRefs,
				ReasoningSteps: steps,
			}, nil
		}

		// Extract thought
		thoughtContent := extractThought(content)
		if thoughtContent != "" {
			stepNum++
			steps = append(steps, ReasoningStep{
				Step:      stepNum,
				Type:      "thought",
				Content:   thoughtContent,
				Timestamp: now,
			})
		}

		// Extract action and action input
		actionName, actionInputRaw := extractAction(content)
		if actionName == "" {
			// Cannot parse output → return error for fallback
			preview := content
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			return nil, fmt.Errorf("react: cannot parse LLM output at iteration %d: %q", i, preview)
		}

		var actionInput map[string]interface{}
		if actionInputRaw != "" {
			if err := json.Unmarshal([]byte(actionInputRaw), &actionInput); err != nil {
				// Try to build a minimal input with the raw string as query
				actionInput = map[string]interface{}{"query": actionInputRaw}
			}
		}

		stepNum++
		steps = append(steps, ReasoningStep{
			Step:        stepNum,
			Type:        "action",
			Content:     actionName,
			ActionInput: actionInput,
			Timestamp:   now,
		})

		// Execute tool
		tool, ok := e.config.Registry.Get(actionName)
		var observation string
		if !ok {
			observation = fmt.Sprintf("Error: tool %q not found", actionName)
		} else {
			result, execErr := tool.Execute(ctx, actionInput)
			if execErr != nil {
				observation = fmt.Sprintf("Error: %v", execErr)
			} else {
				observation, allRefs = formatToolResult(result, allRefs)
			}
		}

		stepNum++
		steps = append(steps, ReasoningStep{
			Step:      stepNum,
			Type:      "observation",
			Content:   observation,
			Timestamp: now,
		})

		// Append assistant message and observation to history
		messages = append(messages,
			&schema.Message{Role: schema.Assistant, Content: content},
			&schema.Message{Role: schema.User, Content: "Observation: " + observation},
		)
	}

	// Max iterations reached: ask LLM to summarize
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: "Please summarize your findings and provide a final answer based on what you have gathered so far.",
	})
	resp, err := e.config.Model.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("react: final summary generate failed: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	answer := strings.TrimSpace(resp.Content)
	if idx := strings.Index(answer, "Final Answer:"); idx >= 0 {
		answer = strings.TrimSpace(answer[idx+len("Final Answer:"):])
	}
	stepNum++
	steps = append(steps, ReasoningStep{
		Step:      stepNum,
		Type:      "final_answer",
		Content:   answer,
		Timestamp: now,
	})

	return &ReactResult{
		Answer:         answer,
		References:     allRefs,
		ReasoningSteps: steps,
	}, nil
}

// extractThought extracts the content after "Thought:" in the LLM output.
func extractThought(content string) string {
	if idx := strings.Index(content, "Thought:"); idx >= 0 {
		rest := content[idx+len("Thought:"):]
		// Take up to the next recognized keyword
		end := len(rest)
		for _, kw := range []string{"Action:", "Final Answer:"} {
			if i := strings.Index(rest, kw); i >= 0 && i < end {
				end = i
			}
		}
		return strings.TrimSpace(rest[:end])
	}
	return ""
}

// extractAction extracts the tool name and raw JSON input from the LLM output.
func extractAction(content string) (name string, inputRaw string) {
	actionIdx := strings.Index(content, "Action:")
	if actionIdx < 0 {
		return "", ""
	}
	rest := content[actionIdx+len("Action:"):]

	// Action name is until newline
	newline := strings.Index(rest, "\n")
	if newline < 0 {
		return strings.TrimSpace(rest), ""
	}
	name = strings.TrimSpace(rest[:newline])
	rest = rest[newline+1:]

	// Action Input follows
	inputIdx := strings.Index(rest, "Action Input:")
	if inputIdx < 0 {
		return name, ""
	}
	inputRest := strings.TrimSpace(rest[inputIdx+len("Action Input:"):])
	// Take until the next blank line or end of string
	lines := strings.SplitN(inputRest, "\n\n", 2)
	inputRaw = strings.TrimSpace(lines[0])
	return name, inputRaw
}

// formatToolResult converts a tool result to an observation string and appends any documents to refs.
func formatToolResult(result interface{}, existing []*schema.Document) (string, []*schema.Document) {
	// Use JSON marshaling for a generic string representation
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("%v", result), existing
	}

	// Try to extract documents from the result
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err == nil {
		if docsRaw, ok := raw["documents"]; ok {
			docsData, _ := json.Marshal(docsRaw)
			var docs []*schema.Document
			if err := json.Unmarshal(docsData, &docs); err == nil {
				existing = append(existing, docs...)
				// Build a concise summary
				var sb strings.Builder
				fmt.Fprintf(&sb, "Found %d documents.", len(docs))
				for i, d := range docs {
					if i >= 3 {
						break
					}
					content := d.Content
					if len(content) > 200 {
						content = content[:200] + "..."
					}
					fmt.Fprintf(&sb, "\n[%d] %s", i+1, content)
				}
				return sb.String(), existing
			}
		}
	}

	return string(data), existing
}
