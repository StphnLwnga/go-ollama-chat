// AI Chat — Project 01 frontend.
//
// Same base concepts we built earlier: fetch() POST + ReadableStream reader +
// SSE parsing (split on \n\n, strip "data: ", JSON.parse, watch for [DONE]).
// Re-skinned to look like AI Elements, plus two behaviors that complete that look:
//   • Send/Stop button via AbortController       (doc Extension 2)
//   • Markdown rendering on the completed reply   (doc Extension 4)

const form      = document.getElementById("chat-form");
const input     = document.getElementById("message");
const send      = document.getElementById("send");
const messages  = document.getElementById("messages");
const scroll     = document.getElementById("scroll");
const scrollBtn = document.getElementById("scrollBtn");
const empty     = document.getElementById("empty");

let isStreaming = false;
let controller  = null; // AbortController for the in-flight request

// ── Scroll helpers ──────────────────────────────────────────────────────
const nearBottom = () =>
    scroll.scrollHeight - scroll.scrollTop - scroll.clientHeight < 80;
const scrollToBottom = () => { scroll.scrollTop = scroll.scrollHeight; };

// ── Message builders ────────────────────────────────────────────────────
function addUserMessage(text) {
    empty.hidden = true;
    const el = document.createElement("div");
    el.className = "msg user";
    const bubble = document.createElement("div");
    bubble.className = "bubble";
    bubble.textContent = text;          // textContent = safe, no HTML injection
    el.appendChild(bubble);
    messages.appendChild(el);
    scrollToBottom();
}

function addAssistantMessage() {
    const el = document.createElement("div");
    el.className = "msg assistant";
    el.innerHTML = '<div class="avatar">AI</div>';
    const content  = document.createElement("div");
    content.className = "content";
    const response = document.createElement("div");
    response.className = "response";
    const cursor   = document.createElement("span");
    cursor.className = "cursor";
    response.appendChild(cursor);
    content.appendChild(response);
    el.appendChild(content);
    messages.appendChild(el);
    scrollToBottom();
    return { response, cursor };
}

function setStreaming(on) {
    isStreaming = on;
    send.dataset.state = on ? "streaming" : "ready";
    send.setAttribute("aria-label", on ? "Stop" : "Send");
}

// Extension 4: render markdown ONLY on the finished text, and sanitize it
// (turning model output into HTML is an XSS surface — never trust it raw).
function renderMarkdown(response, text) {
    response.classList.add("rendered");
    response.innerHTML = DOMPurify.sanitize(marked.parse(text));
}

// ── Auto-grow textarea + Enter-to-send ──────────────────────────────────
function autogrow() {
    input.style.height = "auto";
    input.style.height = Math.min(input.scrollHeight, 180) + "px";
}
input.addEventListener("input", autogrow);
input.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); form.requestSubmit(); }
});

// ── Scroll-to-bottom button ─────────────────────────────────────────────
scroll.addEventListener("scroll", () => { scrollBtn.hidden = nearBottom(); });
scrollBtn.addEventListener("click", () =>
    scroll.scrollTo({ top: scroll.scrollHeight, behavior: "smooth" }));

// While streaming, a click on the button means STOP (Extension 2).
send.addEventListener("click", (e) => {
    if (isStreaming) { e.preventDefault(); controller?.abort(); }
});

// ── Submit → stream the reply ───────────────────────────────────────────
form.addEventListener("submit", async (e) => {
    e.preventDefault();
    if (isStreaming) return;            // Enter does nothing mid-stream (Stop is the button)

    const text = input.value.trim();
    if (!text) return;

    addUserMessage(text);
    input.value = "";
    autogrow();

    const { response, cursor } = addAssistantMessage();
    controller = new AbortController();
    setStreaming(true);

    let full = "";
    try {
        const res = await fetch("/chat", {
            method:  "POST",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body:    new URLSearchParams({ message: text }),
            signal:  controller.signal,
        });

        const reader  = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        while (true) {
            const { value, done } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const events = buffer.split("\n\n");
            buffer = events.pop();      // keep the last, possibly-incomplete event

            for (const evt of events) {
                if (!evt.startsWith("data: ")) continue;
                const chunk = JSON.parse(evt.slice(6));

                if (chunk === "[DONE]")  { if (full) renderMarkdown(response, full); cursor.remove(); return; }
                if (chunk === "[ERROR]") { cursor.remove(); response.append(" ⚠️ (stream error)"); return; }

                full += chunk;
                cursor.before(document.createTextNode(chunk)); // safe plain text while streaming
                if (nearBottom()) scrollToBottom();
            }
        }
        // Stream closed without an explicit [DONE].
        if (full) renderMarkdown(response, full);
        cursor.remove();
    } catch (err) {
        cursor.remove();
        if (err.name === "AbortError") {
            if (full) renderMarkdown(response, full); // user stopped — keep what we got
        } else {
            response.append(" ⚠️ (" + err.message + ")");
        }
    } finally {
        setStreaming(false);
        controller = null;
        input.focus();
    }
});
