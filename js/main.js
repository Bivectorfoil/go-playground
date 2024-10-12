(function() {
    const { createApp, ref, onMounted, onUnmounted } = Vue;

    createApp({
        components: {
            EditorComponent: window.EditorComponent
        },
        setup() {
            const code = ref('package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("Hello, world!")\n}');
            const output = ref('');
            let socket = null;

            onMounted(() => {
                socket = new WebSocket('ws://localhost:8080/ws');
                
                socket.onmessage = (event) => {
                    const message = JSON.parse(event.data);
                    if (message.type === 'output') {
                        output.value += message.data;
                    } else if (message.type === 'error') {
                        output.value += 'Error: ' + message.data + '\n';
                    } else if (message.type === 'clear') {
                        output.value = '';
                    }
                };

                socket.onerror = (error) => {
                    console.error('WebSocket error:', error);
                    output.value += 'Connection error, please refresh the page and try again\n';
                };
            });

            onUnmounted(() => {
                if (socket) {
                    socket.close();
                }
            });

            function runCode() {
                output.value = 'Running...\n';
                if (socket && socket.readyState === WebSocket.OPEN) {
                    socket.send(code.value);
                } else {
                    output.value += 'Connection not ready, please try again later\n';
                }
            }

            return {
                code,
                output,
                runCode
            };
        }
    }).mount('#app');
})();
