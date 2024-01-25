// adapted from https://github.com/ember-fastboot/ember-cli-fastboot/blob/master/packages/fastboot/src/fastboot-headers.js

class FastBootHeaders {
  headers = {};

  constructor(headers) {
    headers = headers || {};
    for (let header in headers) {
      let value = headers[header];
      if (typeof value === 'string') {
        value = [value];
      }
      this.headers[header.toLowerCase()] = value;
    }
  }

  append(header, value) {
    header = header.toLowerCase();
    if (!this.has(header)) {
      this.headers[header] = [];
    }
    this.headers[header].push(value);
  }

  delete(header) {
    delete this.headers[header.toLowerCase()];
  }

  entries() {
    let entries = [];
    for (let key in this.headers) {
      let values = this.headers[key];
      for (let index = 0; index < values.length; ++index) {
        entries.push([key, values[index]]);
      }
    }
    return entries[Symbol.iterator]();
  }

  get(header) {
    return this.getAll(header)[0] || null;
  }

  getAll(header) {
    return this.headers[header.toLowerCase()] || [];
  }

  has(header) {
    return this.headers[header.toLowerCase()] !== undefined;
  }

  keys() {
    let entries = [];
    for (let key in this.headers) {
      let values = this.headers[key];
      for (let index = 0; index < values.length; ++index) {
        entries.push(key);
      }
    }
    return entries[Symbol.iterator]();
  }

  set(header, value) {
    header = header.toLowerCase();
    this.headers[header] = [value];
  }

  values() {
    let entries = [];
    for (let key in this.headers) {
      let values = this.headers[key];
      for (let index = 0; index < values.length; ++index) {
        entries.push(values[index]);
      }
    }
    return entries[Symbol.iterator]();
  }

  unknownProperty() {
    throw new Error('FastBootHeaders does not support "unknownProperty" operations.');
  }
}
