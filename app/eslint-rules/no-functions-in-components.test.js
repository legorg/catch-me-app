'use strict';

const { describe, it } = require('node:test');
const { RuleTester } = require('eslint');
const tsParser = require('@typescript-eslint/parser');
const rule = require('./no-functions-in-components');

// RuleTester integrates with node:test through these hooks.
RuleTester.describe = describe;
RuleTester.it = it;

const tester = new RuleTester({
  languageOptions: {
    parser: tsParser,
    parserOptions: { ecmaFeatures: { jsx: true } },
  },
});

const screen = '/project/src/app/game.tsx';

tester.run('no-functions-in-components', rule, {
  valid: [
    // trivial inline handlers
    { code: `function Foo() { return <Button onPress={() => router.push('/jokers')} />; }` },
    { code: `function Foo() { return <Button onPress={() => setOpen(!open)} />; }` },
    { code: `function Foo() { return <Button onPress={() => { go(1); }} />; }` },
    // effect callbacks and other hook arguments
    { code: `function Foo() { useEffect(() => { doIt(); }, []); return <View />; }` },
    { code: `function Foo() { useLayoutEffect(() => { doIt(); }, []); return <View />; }` },
    { code: `function Foo() { const [a] = useState(() => compute()); return <View />; }` },
    // array method callbacks
    { code: `function Foo({ xs }) { return <View>{xs.map((x) => <Item key={x} x={x} />)}</View>; }` },
    {
      code: `function Foo({ xs }) { const a = xs.filter((x) => x.ok).sort((p, q) => p - q); return <View>{a.length}</View>; }`,
    },
    // custom hooks may define functions
    { code: `function useThing() { const go = () => 1; function other() {} return go; }` },
    { code: `const useThing = () => { const go = useCallback(() => 1, []); return go; };` },
    // non-component helpers (no JSX returned)
    { code: `function helper() { const go = () => 1; return go; }` },
    { code: `function Factory() { const go = () => 1; return go; }` },
    // component-less default export that does not return JSX
    { code: `export default function thing() { const go = () => 1; return go; }`, filename: screen },
    // lowercase non-default-export function returning JSX is not a component
    { code: `function renderRow() { const go = () => 1; return <View />; }` },
    // default export outside src/app and src/components is not detected by default
    { code: `export default function () { const go = () => 1; return <View />; }`, filename: '/project/src/other/x.tsx' },
    // module-level functions are fine
    { code: `const go = () => 1; function Foo() { return <Button onPress={go} />; }` },
    // functions nested in a handler are not reported again; the handler itself is the problem
    { code: `function Foo() { useEffect(() => { run(async () => { await x(); }); }, []); return <View />; }` },
    // option: trivial handlers can be allowed explicitly
    {
      code: `function Foo() { return <Button onPress={() => go()} />; }`,
      options: [{ allowTrivialInlineHandlers: true }],
    },
  ],

  invalid: [
    // nested function declaration
    {
      code: `function Foo() { function handle() {} return <View />; }`,
      errors: [{ messageId: 'nestedDeclaration', data: { name: 'handle', component: 'Foo' } }],
    },
    // const arrow / function expression
    {
      code: `function Foo() { const handle = () => {}; return <View />; }`,
      errors: [{ messageId: 'variableFunction' }],
    },
    {
      code: `const Foo = () => { const handle = function () {}; return <View />; };`,
      errors: [{ messageId: 'variableFunction' }],
    },
    // expression-bodied arrow component
    {
      code: `const Foo = () => <Button onPress={() => { a(); b(); }} />;`,
      errors: [{ messageId: 'inlineHandler' }],
    },
    // async functions assigned to variables
    {
      code: `function Foo() { const load = async () => { await x(); }; return <View />; }`,
      errors: [{ messageId: 'asyncFunction' }],
    },
    {
      code: `function Foo() { const load = async function () {}; return <View />; }`,
      errors: [{ messageId: 'asyncFunction' }],
    },
    // useCallback / useMemo wrapped functions (always reported)
    {
      code: `function Foo() { const go = useCallback(() => 1, []); return <View />; }`,
      errors: [{ messageId: 'hookWrapped', data: { hook: 'useCallback', component: 'Foo' } }],
    },
    {
      code: `function Foo() { const v = useMemo(() => { return 1; }, []); return <View />; }`,
      errors: [{ messageId: 'hookWrapped' }],
    },
    {
      code: `function Foo() { const go = React.useCallback(() => 1, []); return <View />; }`,
      errors: [{ messageId: 'hookWrapped' }],
    },
    // helper defined inside an effect callback is still defined in the component
    {
      code: `function Foo() { useEffect(() => { const tick = () => {}; tick(); }, []); return <View />; }`,
      errors: [{ messageId: 'variableFunction' }],
    },
    // inline callbacks with logic
    {
      code: `function Foo() { return <Button onPress={() => { a(); b(); }} />; }`,
      errors: [{ messageId: 'inlineHandler' }],
    },
    {
      code: `function Foo() { return <Button onPress={() => (open ? close() : show())} />; }`,
      errors: [{ messageId: 'inlineHandler' }],
    },
    {
      code: `function Foo() { return <Button onPress={() => ok && go()} />; }`,
      errors: [{ messageId: 'inlineHandler' }],
    },
    {
      code: `function Foo() { return <Button onPress={async () => { await go(); }} />; }`,
      errors: [{ messageId: 'inlineHandler' }],
    },
    {
      code: `function Foo() { return <Button onPress={() => go(() => 1)} />; }`,
      errors: [{ messageId: 'inlineHandler' }],
    },
    {
      code: `function Foo() { return <Button onPress={() => { if (x) go(); }} />; }`,
      errors: [{ messageId: 'inlineHandler' }],
    },
    // trivial handlers are reported when the option is off
    {
      code: `function Foo() { return <Button onPress={() => go()} />; }`,
      options: [{ allowTrivialInlineHandlers: false }],
      errors: [{ messageId: 'inlineHandler' }],
    },
    // default export under src/app or src/components, any name
    {
      code: `export default function () { const go = () => 1; return <View />; }`,
      filename: screen,
      errors: [{ messageId: 'variableFunction', data: { name: 'go', component: 'default export' } }],
    },
    {
      code: `export default function game() { const go = () => 1; return <View />; }`,
      filename: '/project/src/components/game.tsx',
      errors: [{ messageId: 'variableFunction', data: { name: 'go', component: 'game' } }],
    },
    // memo / forwardRef wrapped components, conditional JSX returns
    {
      code: `const Foo = memo(() => { const go = () => 1; return cond ? <A /> : null; });`,
      errors: [{ messageId: 'variableFunction' }],
    },
    {
      code: `const Foo = forwardRef(function (props, ref) { if (!x) return null; const go = () => 1; return <A />; });`,
      errors: [{ messageId: 'variableFunction' }],
    },
    // only the outermost definition is reported, not functions nested inside it
    {
      code: `function Foo() { const outer = () => { const inner = () => 1; return inner; }; return <View />; }`,
      errors: [{ messageId: 'variableFunction', data: { name: 'outer', component: 'Foo' } }],
    },
  ],
});
