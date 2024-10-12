(function() {
  const { defineComponent, onMounted, ref } = Vue;

  const EditorComponent = defineComponent({
    name: 'EditorComponent',
    props: {
      initialValue: {
        type: String,
        default: 'package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("Hello, world!")\n}'
      }
    },
    emits: ['update:modelValue'],
    setup(props, { emit }) {
      const editorRef = ref(null);
      let editor;

      onMounted(() => {
        editor = CodeMirror(editorRef.value, {
          value: props.initialValue,
          mode: 'go',
          theme: 'monokai',
          lineNumbers: true,
          indentUnit: 4,
          tabSize: 4,
          indentWithTabs: true,
          autofocus: true
        });

        editor.on('change', () => {
          emit('update:modelValue', editor.getValue());
        });
      });

      const getValue = () => editor.getValue();
      const setValue = (value) => editor.setValue(value);

      return {
        editorRef,
        getValue,
        setValue
      };
    },
    template: '<div ref="editorRef"></div>'
  });

  window.EditorComponent = EditorComponent;
})();
