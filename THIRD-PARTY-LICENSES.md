# Third-Party Licenses

This document contains the licenses for third-party software used in the Licet project.

## Embedded Frontend Libraries

All frontend libraries embedded in `web/static/` are licensed under the MIT License, which is compatible with Licet's GPL-3.0 license.

---

### Bootstrap v5.3.3

**License:** MIT License

**Copyright:** 2011-2024 The Bootstrap Authors

**URL:** https://getbootstrap.com/

**Files:**
- `web/static/css/bootstrap.min.css`
- `web/static/js/bootstrap.min.js`

**License Text:**

```
The MIT License (MIT)

Copyright (c) 2011-2024 The Bootstrap Authors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

---

### Chart.js v4.4.1

**License:** MIT License

**Copyright:** 2023 Chart.js Contributors

**URL:** https://www.chartjs.org

**Files:**
- `web/static/js/chart.min.js`

**License Text:**

```
The MIT License (MIT)

Copyright (c) 2023 Chart.js Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

---

### chartjs-chart-matrix v2.0.1

**License:** MIT License

**Copyright:** 2023 Jukka Kurkela

**URL:** https://chartjs-chart-matrix.pages.dev/

**Files:**
- `web/static/js/chartjs-chart-matrix.min.js`

**License Text:**

```
The MIT License (MIT)

Copyright (c) 2023 Jukka Kurkela

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

---

### chartjs-adapter-date-fns v3.0.0

**License:** MIT License

**Copyright:** 2022 chartjs-adapter-date-fns Contributors

**URL:** https://www.chartjs.org

**Files:**
- `web/static/js/chartjs-adapter-date-fns.bundle.min.js`

**License Text:**

```
The MIT License (MIT)

Copyright (c) 2022 chartjs-adapter-date-fns Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

---

## Go Dependencies

For a complete list of Go module dependencies and their licenses, see `go.mod`. All dependencies are compatible with GPL-3.0.

Key Go dependencies include:
- **github.com/go-chi/chi/v5** - MIT License
- **github.com/go-chi/cors** - MIT License
- **github.com/jmoiron/sqlx** - MIT License
- **github.com/lib/pq** - MIT License
- **github.com/mattn/go-sqlite3** - MIT License
- **github.com/robfig/cron/v3** - MIT License
- **github.com/sirupsen/logrus** - MIT License
- **github.com/spf13/viper** - MIT License
- **gopkg.in/gomail.v2** - MIT License

---

## License Compatibility

All third-party libraries used in this project are licensed under the MIT License, which is compatible with the GNU General Public License v3.0. When these MIT-licensed components are distributed as part of Licet, the combined work is distributed under the GPL-3.0 terms, while the original MIT-licensed components retain their MIT license.

For more information about license compatibility, see:
- https://www.gnu.org/licenses/license-list.html#Expat
- https://www.gnu.org/licenses/gpl-faq.html#AllCompatibility
