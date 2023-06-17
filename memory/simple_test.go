package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimple_MemoryVariables(t *testing.T) {
	t.Run("MemoryVariables", func(t *testing.T) {
		t.Run("Empty", func(t *testing.T) {
			memory := NewSimple()

			variables := memory.MemoryVariables()

			assert.Empty(t, variables, "Memory variables should be empty when no memories are stored")
		})

		t.Run("NonEmpty", func(t *testing.T) {
			memory := NewSimple()
			memory.memories["name"] = "John"
			memory.memories["age"] = 30

			expectedVariables := []string{"name", "age"}

			variables := memory.MemoryVariables()

			assert.ElementsMatch(t, expectedVariables, variables, "Memory variables should match the expected variables")
		})
	})

	t.Run("LoadMemoryVariables", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			memory := NewSimple()
			memory.memories["name"] = "John"
			memory.memories["age"] = 30

			inputs := map[string]interface{}{
				"var1": "value1",
				"var2": "value2",
			}

			expectedOutputs := map[string]interface{}{
				"name": "John",
				"age":  30,
			}

			outputs, err := memory.LoadMemoryVariables(inputs)

			assert.NoError(t, err, "LoadMemoryVariables should not return an error")
			assert.Equal(t, expectedOutputs, outputs, "Loaded memory variables should match the expected outputs")
		})
	})

	t.Run("SaveContext", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			memory := NewSimple()

			inputs := map[string]interface{}{
				"name": "John",
				"age":  30,
			}

			outputs := map[string]interface{}{
				"var1": "value1",
				"var2": "value2",
			}

			err := memory.SaveContext(inputs, outputs)

			assert.NoError(t, err, "SaveContext should not return an error")
			// No assertions made as SaveContext does not modify the state of the Simple memory.
		})
	})

	t.Run("Clear", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			memory := NewSimple()

			err := memory.Clear()

			assert.NoError(t, err, "Clear should not return an error")
			// No assertions made as Clear does not modify the state of the Simple memory.
		})
	})
}