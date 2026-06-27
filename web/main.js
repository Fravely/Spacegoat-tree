const { createApp } = Vue;

createApp({
  data() {
    return {
      root: null,
      stats: { size: 0, maxSize: 0, height: -1, rebuilds: 0 },
      mode: "memoria",
      inOrder: [],
      targetID: 105,
      activeNodes: {},
      stepText: "",
      foundProduct: "",
      message: "Listo para simular inserciones, busquedas y eliminaciones.",
      isError: false,
      form: {
        id: 109,
        name: "Producto nuevo",
        category: "Demo",
        price: 99.9,
        stock: 5,
      },
    };
  },
  computed: {
    treeHtml() {
      return renderTree(this.root, this.activeNodes);
    },
  },
  async mounted() {
    await this.loadTree();
  },
  methods: {
    async request(path, options = {}) {
      const response = await fetch(path, {
        headers: { "Content-Type": "application/json" },
        ...options,
      });
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.message || "Error inesperado");
      }
      return data;
    },
    async loadTree() {
      const data = await this.request("/api/tree");
      this.root = data.root;
      this.stats = data.stats;
      this.mode = data.mode || "memoria";
      this.inOrder = data.inOrder || [];
    },
    async insertProduct() {
      try {
        const product = {
          ID: Number(this.form.id),
          Name: this.form.name,
          Category: this.form.category,
          Price: Number(this.form.price),
          Stock: Number(this.form.stock),
        };
        const data = await this.request("/api/products", {
          method: "POST",
          body: JSON.stringify(product),
        });
        this.say(data.message);
        this.targetID = product.ID;
        await this.loadTree();
        if (data.trace?.updated) {
          await this.playPath(data.trace.path, "Buscando el producto que se va a actualizar...");
          this.flashNodes([product.ID], "update");
          return;
        }
        await this.playInsertTrace(data.trace, product.ID);
      } catch (error) {
        this.say(error.message, true);
      }
    },
    async searchProduct() {
      try {
        const product = await this.request(`/api/products/${this.targetID}`);
        this.foundProduct = JSON.stringify(product, null, 2);
        this.say(`Producto ${this.targetID} encontrado`);
        await this.playPath(findPath(this.root, this.targetID), "Buscando por ID...");
        this.flashNodes([this.targetID], "search");
      } catch (error) {
        this.foundProduct = "";
        this.say(error.message, true);
      }
    },
    async deleteProduct() {
      try {
        const data = await this.request(`/api/products/${this.targetID}`, {
          method: "DELETE",
        });
        this.foundProduct = "";
        this.say(data.message);
        await this.playPath(findPath(this.root, this.targetID), "Recorriendo el arbol antes de eliminar...");
        this.stepText = `Eliminando el nodo ${this.targetID}.`;
        this.flashNodes([this.targetID], "delete", 520);
        await this.wait(520);
        await this.loadTree();
        this.clearAnimation();
      } catch (error) {
        this.say(error.message, true);
      }
    },
    async resetTree() {
      try {
        const data = await this.request("/api/reset", { method: "POST" });
        this.foundProduct = "";
        this.clearAnimation();
        this.say(data.message);
        await this.loadTree();
      } catch (error) {
        this.say(error.message, true);
      }
    },
    say(message, isError = false) {
      this.message = message;
      this.isError = isError;
    },
    setAnimation(ids, type) {
      const next = {};
      for (const id of ids || []) {
        next[Number(id)] = type;
      }
      this.activeNodes = next;
    },
    clearAnimation() {
      this.activeNodes = {};
      this.stepText = "";
    },
    flashNodes(ids, type, duration = 1400) {
      this.setAnimation(ids, type);
      window.setTimeout(() => {
        this.clearAnimation();
      }, duration);
    },
    async playPath(path, text) {
      const steps = path || [];
      for (const id of steps) {
        this.stepText = `${text} Visitando nodo ${id}.`;
        this.setAnimation([id], "visit");
        await this.wait(520);
      }
    },
    async playInsertTrace(trace, insertedID) {
      if (!trace) {
        this.flashNodes([insertedID], "insert");
        return;
      }
      await this.playPath(trace.path, `Insertando ${insertedID}.`);

      this.stepText = `Nodo ${insertedID} insertado. Profundidad: ${trace.depth}, limite permitido: ${trace.maxDepth}.`;
      this.setAnimation([insertedID], "insert");
      await this.wait(950);

      if (!trace.scapegoat) {
        this.stepText = "No se necesita reconstruccion: la profundidad esta dentro del limite.";
        await this.wait(1100);
        this.clearAnimation();
        return;
      }

      this.stepText = `Chivo expiatorio detectado: nodo ${trace.scapegoat}.`;
      this.setAnimation([trace.scapegoat], "scapegoat");
      await this.wait(1200);

      this.stepText = `Reconstruyendo el subarbol con nodos: ${(trace.rebuiltKeys || []).join(", ")}.`;
      this.setAnimation(trace.rebuiltKeys || [], "rebuild");
      await this.wait(1500);
      this.clearAnimation();
    },
    wait(ms) {
      return new Promise((resolve) => window.setTimeout(resolve, ms));
    },
  },
}).mount("#app");

function renderTree(node, activeNodes) {
  if (!node) {
    return `<span class="empty">vacio</span>`;
  }

  const name = escapeHTML(shortName(node.value));
  const activeClass = activeNodes?.[Number(node.key)] ? ` is-${activeNodes[Number(node.key)]}` : "";
  const hasChildren = node.left || node.right;
  const children = hasChildren
    ? `<div class="children">
        <div class="child child-left">
          <span class="branch-label">izq</span>
          ${node.left ? renderTree(node.left, activeNodes) : `<span class="empty">vacio</span>`}
        </div>
        <div class="child child-right">
          <span class="branch-label">der</span>
          ${node.right ? renderTree(node.right, activeNodes) : `<span class="empty">vacio</span>`}
        </div>
      </div>`
    : "";

  return `<div class="tree-node">
    <div class="node-card${activeClass}">
      <strong>${node.key}</strong>
      <small>${name}</small>
    </div>
    ${children}
  </div>`;
}

function shortName(value) {
  const name = value?.Name || value?.name || "producto";
  return name.length > 16 ? `${name.slice(0, 15)}...` : name;
}

function findPath(root, targetID) {
  const path = [];
  let current = root;
  const key = Number(targetID);
  while (current) {
    path.push(current.key);
    if (Number(current.key) === key) {
      break;
    }
    current = key < Number(current.key) ? current.left : current.right;
  }
  return path;
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
