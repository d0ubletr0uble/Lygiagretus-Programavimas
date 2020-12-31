'use strict';

const {writeFileSync, appendFileSync} = require('fs');
const endOfLine = require('os').EOL;
const {start, dispatch, spawnStateless, spawn} = require('nact');
const system = start();

// MESSAGE TYPES
const FILTER = 0, COLLECT = 1, STORE = 2, RETRIEVE = 3, PRINT = 4, END_OF_DATA = 5

// PROGRAM SETTINGS
const WORKER_COUNT = 4;
const FILTER_CRITERIA = 26;
const DATA_FILE = './IFK-8_OdinasT_dat_1.json';
const RESULT_FILE = 'IFK-8_OdinasT_rez.txt';

/**
 * Checks if car meets filter criteria and if it does then sends it back to distributor.
 */
function filter(car, ctx) {
    car.age = new Date().getFullYear() - car.year + Math.trunc(car.mileage / 20_000);
    if (car.age < FILTER_CRITERIA) // send back to distributor which sends to collector
        dispatch(ctx.parent, {type: COLLECT, data: car});
}

/**
 * Stores sorted cars in state until distributor asks to retrieve them.
 */
function collect(state = [], msg, ctx) {
    switch (msg.type) {
        case STORE: // got data from distributor
            state.push(msg.data);
            // sort by car age ascending, and then by year descending
            state.sort((a, b) => a.age > b.age ? 1 : ((a.age === b.age && a.year < b.year) ? 1 : -1));
            break;
        case RETRIEVE: // distributor asks for data back
            dispatch(ctx.parent, {type: PRINT, data: state})
            break;
    }
    return state;
}

/**
 * Writes car data to result file.
 */
function writeToFile(cars, _) {
    // write input data
    writeFileSync(RESULT_FILE, [
        '='.repeat(32),
        '|          INPUT DATA          |',
        '='.repeat(32),
        '|     Make    | Year|  Mileage |',
        '='.repeat(32),
        ...require(DATA_FILE).map(car => `|${car.make.padEnd(13)}| ${car.year}|${car.mileage.toString().padStart(10)}|`),
        '='.repeat(32)].join(endOfLine));

    // append output data
    appendFileSync(RESULT_FILE, [
        endOfLine.repeat(2),
        '='.repeat(36),
        '|            OUTPUT DATA           |',
        '='.repeat(36),
        '|     Make    | Year|  Mileage |Age|',
        '='.repeat(36),
        ...cars.map(car => `|${car.make.padEnd(13)}| ${car.year}|${car.mileage.toString().padStart(10)}| ${car.age}|`),
        '='.repeat(36)].join(endOfLine));
}

/**
 * Creates actors and then sends and receives data from them.
 */
function distribute(state, msg, ctx) {
    // initialize actors on first call and store them in state
    if (state === undefined) {
        state = {
            workers: [...Array(WORKER_COUNT).keys()].map(
                id => spawnStateless(ctx.self, filter, `worker_${id}`)
            ),
            nextWorkerIndex: 0,
            collector: spawn(ctx.self, collect, `collector`),
            printer: spawnStateless(ctx.self, writeToFile, `printer`)
        };
    }

    // proceed distributing messages between actors
    switch (msg.type) {
        case FILTER: // data received -> give it to next worker
            dispatch(state.workers[state.nextWorkerIndex], msg.data);
            // on next message give work to another worker
            state.nextWorkerIndex = (state.nextWorkerIndex + 1) % state.workers.length;
            break;
        case COLLECT: // worker returned data -> send it to collector
            dispatch(state.collector, {type: STORE, data: msg.data});
            break;
        case PRINT: // got data back from collector -> print it
            dispatch(state.printer, msg.data)
            break;
        case END_OF_DATA: // all data is transmitted -> retrieve processed data back from collector
            dispatch(state.collector, {type: RETRIEVE});
            break;
    }
    return state;
}

// program entry point
if (require.main === module) {
    const distributor = spawn(system, distribute, 'distributor');

    // array of 30 cars in datafile
    require(DATA_FILE).forEach(car => dispatch(distributor, {type: FILTER, data: car}));
    dispatch(distributor, {type: END_OF_DATA});
}
