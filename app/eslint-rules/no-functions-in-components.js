'use strict';

/**
 * catchme/no-functions-in-components
 *
 * Reports functions defined inside React components so that logic lives in
 * custom hooks (`useXxx`) or module-level helpers instead.
 *
 * What is reported (when the function is defined directly in a component body,
 * looking through effect callbacks, see below):
 *  - nested function declarations              function handle() {}
 *  - functions assigned to variables           const handle = () => {} / function () {}
 *    (async ones get a dedicated message)
 *  - functions passed to useCallback/useMemo   (ALL of them, regardless of size;
 *    simple and predictable: move the logic to a custom hook)
 *  - inline JSX attribute callbacks that contain logic, e.g. multiple statements,
 *    branches (if/ternary/&&/||/??), loops, try/catch, await or nested functions.
 *
 * What is allowed:
 *  - trivial inline JSX callbacks: a single call expression with no logic,
 *    e.g. `onPress={() => router.push('/x')}` (option allowTrivialInlineHandlers)
 *  - the callback of useEffect/useLayoutEffect/useInsertionEffect, and arguments
 *    of any other hook (e.g. lazy `useState(() => ...)` initializers, `useRef`).
 *    NOTE: effect callbacks are "transparent": a function declared inside an
 *    effect callback is still a function defined in the component and is reported.
 *  - callbacks of array methods (map, filter, find, every, some, reduce, sort, ...)
 *  - any other callback passed straight to a call (e.g. `.then(...)`,
 *    `setTimeout(...)`); only variable-assigned / declared / JSX-attribute /
 *    useCallback / useMemo functions are reported.
 *  - custom hooks (useXxx) and non-component functions are never checked.
 *  - functions nested inside another reported/unrelated function (e.g. inside a
 *    handler) are not reported again; only the outermost definition is.
 *
 * Component detection (heuristic, no type info):
 *  A function is a component when it returns JSX (a JSXElement/JSXFragment, also
 *  through ternaries and `&&`/`||`) AND either
 *   - its name starts with an uppercase letter (function declaration/expression
 *     id, the variable it is assigned to, also when wrapped in memo/forwardRef), or
 *   - it is the default export of a file under src/app/ or src/components/
 *     (option defaultExportDirs).
 *  Functions named useXxx are never components.
 */

const EFFECT_HOOKS = new Set(['useEffect', 'useLayoutEffect', 'useInsertionEffect']);
const WRAPPER_HOOKS = new Set(['useCallback', 'useMemo']);
const COMPONENT_WRAPPERS = new Set(['memo', 'forwardRef']);

// AST node types that count as "logic" inside an otherwise trivial inline handler.
const LOGIC_NODE_TYPES = new Set([
  'ConditionalExpression',
  'LogicalExpression',
  'AwaitExpression',
  'IfStatement',
  'SwitchStatement',
  'ForStatement',
  'ForInStatement',
  'ForOfStatement',
  'WhileStatement',
  'DoWhileStatement',
  'TryStatement',
  'ArrowFunctionExpression',
  'FunctionExpression',
  'FunctionDeclaration',
]);

const isFunction = (node) =>
  node.type === 'FunctionDeclaration' ||
  node.type === 'FunctionExpression' ||
  node.type === 'ArrowFunctionExpression';

/** Name of the called function: `foo()` -> foo, `React.foo()` -> foo. */
function calleeName(call) {
  const callee = call.callee;
  if (callee.type === 'Identifier') return callee.name;
  if (callee.type === 'MemberExpression' && !callee.computed && callee.property.type === 'Identifier') {
    return callee.property.name;
  }
  return null;
}

const isHookName = (name) => typeof name === 'string' && /^use[A-Z0-9]/.test(name);
const isUpperName = (name) => typeof name === 'string' && /^[A-Z]/.test(name);

/** Does this expression evaluate to JSX (directly, or via ternary / && / ||)? */
function isJsxExpression(node) {
  if (!node) return false;
  switch (node.type) {
    case 'JSXElement':
    case 'JSXFragment':
      return true;
    case 'ConditionalExpression':
      return isJsxExpression(node.consequent) || isJsxExpression(node.alternate);
    case 'LogicalExpression':
      return isJsxExpression(node.left) || isJsxExpression(node.right);
    case 'TSAsExpression':
    case 'TSNonNullExpression':
    case 'TSSatisfiesExpression':
      return isJsxExpression(node.expression);
    default:
      return false;
  }
}

/** The name a function is known by, looking through memo()/forwardRef() wrappers. */
function functionName(node) {
  if (node.id) return node.id.name;
  let parent = node.parent;
  while (
    parent &&
    parent.type === 'CallExpression' &&
    COMPONENT_WRAPPERS.has(calleeName(parent))
  ) {
    parent = parent.parent;
  }
  if (parent && parent.type === 'VariableDeclarator' && parent.id.type === 'Identifier') {
    return parent.id.name;
  }
  return null;
}

/** Is the function (possibly wrapped in memo/forwardRef) the default export? */
function isDefaultExport(node) {
  if (node.parent && node.parent.type === 'ExportDefaultDeclaration') return true;
  let parent = node.parent;
  while (parent && parent.type === 'CallExpression' && COMPONENT_WRAPPERS.has(calleeName(parent))) {
    parent = parent.parent;
  }
  return parent !== node.parent && parent && parent.type === 'ExportDefaultDeclaration';
}

/** If `node` is a call argument, return that call. */
function enclosingCall(node) {
  const parent = node.parent;
  return parent && parent.type === 'CallExpression' && parent.arguments.includes(node) ? parent : null;
}

module.exports = {
  meta: {
    type: 'suggestion',
    docs: {
      description:
        'Disallow defining functions inside React components; move logic into custom hooks or helpers.',
    },
    schema: [
      {
        type: 'object',
        properties: {
          allowTrivialInlineHandlers: { type: 'boolean' },
          defaultExportDirs: { type: 'array', items: { type: 'string' } },
        },
        additionalProperties: false,
      },
    ],
    messages: {
      nestedDeclaration:
        "Function '{{name}}' is declared inside component '{{component}}'. Move it to a custom hook or a module-level helper.",
      variableFunction:
        "Function '{{name}}' is defined inside component '{{component}}'. Move it to a custom hook or a module-level helper.",
      asyncFunction:
        "Async function '{{name}}' is defined inside component '{{component}}'. Move it to a custom hook or a module-level helper.",
      hookWrapped:
        "'{{hook}}' wraps a function inside component '{{component}}'. Move this logic (and the {{hook}} call) into a custom hook.",
      inlineHandler:
        "Inline callback in component '{{component}}' contains logic. Only a single call expression is allowed inline; move the rest to a custom hook.",
    },
  },

  create(context) {
    const options = context.options[0] || {};
    const allowTrivial = options.allowTrivialInlineHandlers !== false;
    const dirs = options.defaultExportDirs || ['src/app/', 'src/components/'];
    const filename = context.filename.replace(/\\/g, '/');
    const inDefaultExportDir = dirs.some((d) => filename.includes(`/${d}`) || filename.startsWith(d));
    const sourceCode = context.sourceCode;

    // One frame per function seen; frames form a tree through `parent`.
    const frames = [];
    const stack = [];

    function enter(node) {
      const frame = {
        node,
        parent: stack[stack.length - 1] || null,
        returnsJsx: node.type === 'ArrowFunctionExpression' && isJsxExpression(node.body),
      };
      frames.push(frame);
      stack.push(frame);
    }
    function exit() {
      stack.pop();
    }

    function isComponent(frame) {
      if (!frame.returnsJsx) return false;
      const name = functionName(frame.node);
      if (isHookName(name)) return false;
      if (isUpperName(name)) return true;
      return inDefaultExportDir && isDefaultExport(frame.node);
    }

    /** Effect callbacks are transparent: we keep looking for the component above them. */
    function isEffectCallback(node) {
      const call = enclosingCall(node);
      return call !== null && EFFECT_HOOKS.has(calleeName(call));
    }

    /** Find the component a function is (directly) defined in, if any. */
    function owningComponent(frame) {
      for (let f = frame.parent; f; f = f.parent) {
        if (isComponent(f)) return f;
        if (!isEffectCallback(f.node)) return null;
      }
      return null;
    }

    /** True when a node contains no logic (see LOGIC_NODE_TYPES). */
    function hasLogic(node) {
      if (LOGIC_NODE_TYPES.has(node.type)) return true;
      const keys = sourceCode.visitorKeys[node.type] || [];
      return keys.some((key) => {
        const child = node[key];
        const list = Array.isArray(child) ? child : [child];
        return list.some((c) => c && typeof c.type === 'string' && hasLogic(c));
      });
    }

    /** `() => call(...)`, `() => { call(...); }` or `() => { return call(...); }`, without logic. */
    function isTrivialHandler(fn) {
      if (fn.async) return false;
      let expr = fn.body;
      if (expr.type === 'BlockStatement') {
        if (expr.body.length !== 1) return false;
        const only = expr.body[0];
        if (only.type === 'ExpressionStatement') expr = only.expression;
        else if (only.type === 'ReturnStatement' && only.argument) expr = only.argument;
        else return false;
      }
      return expr.type === 'CallExpression' && !hasLogic(expr);
    }

    function check(frame, componentName) {
      const node = frame.node;
      const parent = node.parent;
      const report = (messageId, data) =>
        context.report({ node, messageId, data: { component: componentName, ...data } });

      if (node.type === 'FunctionDeclaration') {
        report('nestedDeclaration', { name: node.id ? node.id.name : 'anonymous' });
        return;
      }

      if (parent.type === 'VariableDeclarator' && parent.init === node) {
        const name = parent.id.type === 'Identifier' ? parent.id.name : 'anonymous';
        report(node.async ? 'asyncFunction' : 'variableFunction', { name });
        return;
      }

      const call = enclosingCall(node);
      if (call) {
        const hook = calleeName(call);
        if (WRAPPER_HOOKS.has(hook)) report('hookWrapped', { hook });
        // Effects, other hooks, array methods and other call arguments are allowed.
        return;
      }

      if (parent.type === 'JSXExpressionContainer' && parent.parent.type === 'JSXAttribute') {
        if (allowTrivial && isTrivialHandler(node)) return;
        report('inlineHandler');
      }
    }

    return {
      FunctionDeclaration: enter,
      FunctionExpression: enter,
      ArrowFunctionExpression: enter,
      'FunctionDeclaration:exit': exit,
      'FunctionExpression:exit': exit,
      'ArrowFunctionExpression:exit': exit,

      ReturnStatement(node) {
        const top = stack[stack.length - 1];
        if (top && isJsxExpression(node.argument)) top.returnsJsx = true;
      },

      'Program:exit'() {
        for (const frame of frames) {
          const owner = owningComponent(frame);
          if (owner) check(frame, functionName(owner.node) || 'default export');
        }
      },
    };
  },
};
