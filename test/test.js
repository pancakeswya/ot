const fs = require('fs');
const path = require('path');

// Set up global environment for WebAssembly
globalThis.require = require;
globalThis.fs = fs;
globalThis.TextEncoder = require('util').TextEncoder;
globalThis.TextDecoder = require('util').TextDecoder;
globalThis.performance = require('perf_hooks').performance;
globalThis.crypto = require('crypto');

// Load Go WebAssembly runtime
require('../public/wasm_exec.js');

async function runTests() {
    const go = new Go();
    
    try {
        // Load and instantiate WebAssembly module
        const wasmBuffer = fs.readFileSync(path.join(__dirname, '../public/main.wasm'));
        const wasmModule = await WebAssembly.instantiate(wasmBuffer, go.importObject);
        
        // Start Go program
        go.run(wasmModule.instance);
        
        console.log('Testing OT operations...\n');
        
        // Create a new sequence
        const seq = NewSequence();
        console.log('Created new sequence');
        
        // Test insert
        seq.insert('Hello ');
        console.log('Inserted "Hello "');
        
        // Test retain
        seq.retain(5);
        console.log('Retained 5 characters');
        
        // Test delete
        seq.delete(2);
        console.log('Deleted 2 characters');
        
        // Test apply
        const result = seq.apply('World!!');
        console.log(`Applied to "World!!": "${result}"`);
        
        // Test isNoop
        console.log(`Is noop: ${seq.isNoop()}`);
        
        // Test compose
        const seq2 = NewSequence();
        seq2.insert('!!!');
        const composed = seq.compose(seq2);
        console.log('Composed with sequence that inserts "!!!"');
        
        // Test transform
        const { aPrime, bPrime } = seq.transform(seq2);
        console.log('Transformed sequences');
        
        // Test invert
        const inverse = seq.invert('World!!');
        console.log('Created inverse sequence');
        
        console.log('\nAll tests completed successfully!');
        
    } catch (error) {
        console.error('Error:', error);
        process.exit(1);
    }
}

// Run the tests
runTests(); 