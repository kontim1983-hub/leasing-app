import React, { useEffect, useState } from 'react';
import { Button, Container, Header, Icon, Image, Message, Segment, Table, Tab } from 'semantic-ui-react';
import axios from 'axios';
import 'semantic-ui-css/semantic.min.css';
import './App.css';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

function App() {
    const [activeTab, setActiveTab] = useState(0);
    const [records, setRecords] = useState([]);
    const [recordsV2, setRecordsV2] = useState([]);
    const [loading, setLoading] = useState(false);
    const [loadingV2, setLoadingV2] = useState(false);
    const [uploading, setUploading] = useState(false);
    const [uploadingV2, setUploadingV2] = useState(false);
    const [error, setError] = useState(null);
    const [errorV2, setErrorV2] = useState(null);
    const [successMessage, setSuccessMessage] = useState(null);
    const [successMessageV2, setSuccessMessageV2] = useState(null);
    const [files, setFiles] = useState([]);
    const [filesV2, setFilesV2] = useState([]);

    const fetchFiles = async () => {
        const res = await fetch(`${API_URL}/api/files`);
        if (!res.ok) return;
        const data = await res.json();
        setFiles(Array.isArray(data) ? data : []);
    };

    const fetchFilesV2 = async () => {
        const res = await fetch(`${API_URL}/api/v2/files`);
        if (!res.ok) return;
        const data = await res.json();
        setFilesV2(Array.isArray(data) ? data : []);
    };

    useEffect(() => {
        if (activeTab === 0) {
            fetchRecords();
            fetchFiles();
        } else {
            fetchRecordsV2();
            fetchFilesV2();
        }
    }, [activeTab]);

    const clearChangedColumns = async () => {
        if (!window.confirm("Вы уверены, что хотите очистить все значения в колонке ChangedColumns?")) return;

        try {
            const response = await fetch(`${API_URL}/api/clear-changed-columns`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
            });
            const data = await response.json();
            alert(data.message || "ChangedColumns очищены!");
            fetchRecords();
        } catch (err) {
            alert("Ошибка: " + err.message);
        }
    };

    const deleteDatabase = async () => {
        if (!window.confirm("ВНИМАНИЕ! Это удалит ВСЕ данные из базы навсегда. Вы уверены?")) return;

        const secondConfirm = window.prompt("Для подтверждения введите слово: УДАЛИТЬ");
        if (secondConfirm !== "УДАЛИТЬ") {
            alert("Операция отменена");
            return;
        }

        try {
            const response = await fetch(`${API_URL}/api/delete-all-records`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ confirm: "delete" }),
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`Ошибка сервера: ${response.status} — ${errorText}`);
            }

            const data = await response.json();
            alert(data.message || "Все записи успешно удалены!");
            setRecords([]);
            setFiles([]);
            setSuccessMessage("База данных очищена");
        } catch (err) {
            console.error(err);
            alert("Ошибка при удалении: " + err.message);
        }
    };

    const fetchRecords = async () => {
        setLoading(true);
        setError(null);
        try {
            const res = await axios.get(`${API_URL}/api/records`);
            setRecords(res.data || []);
        } catch (e) {
            setError(`Ошибка загрузки данных: ${e.message}`);
        } finally {
            setLoading(false);
        }
    };

    const handleFileUpload = async (e) => {
        const file = e.target.files?.[0];
        if (!file) return;

        if (!file.name.endsWith('.xlsx')) {
            setError('Пожалуйста, загрузите файл формата .xlsx');
            return;
        }

        setUploading(true);
        setError(null);
        setSuccessMessage(null);

        const formData = new FormData();
        formData.append('file', file);

        try {
            const res = await axios.post(`${API_URL}/api/upload`, formData, {
                headers: { 'Content-Type': 'multipart/form-data' },
            });

            const processed = res.data || [];

            if (processed.length === 0) {
                setSuccessMessage('Новых изменений не обнаружено');
            } else {
                const newCount = processed.filter(r => r.is_new).length;
                const updatedCount = processed.length - newCount;
                setSuccessMessage(`Обработано записей: ${processed.length} (новых: ${newCount}, обновлённых: ${updatedCount})`);
                fetchRecords();
                fetchFiles();
            }
        } catch (e) {
            setError(`Ошибка загрузки файла: ${e.response?.data || e.message}`);
        } finally {
            setUploading(false);
            e.target.value = '';
        }
    };

    const fetchRecordsV2 = async () => {
        setLoadingV2(true);
        setErrorV2(null);
        try {
            const res = await axios.get(`${API_URL}/api/v2/records`);
            setRecordsV2(res.data || []);
        } catch (e) {
            setErrorV2(`Ошибка загрузки данных: ${e.message}`);
        } finally {
            setLoadingV2(false);
        }
    };

    const handleFileUploadV2 = async (e) => {
        const file = e.target.files?.[0];
        if (!file) return;

        if (!file.name.endsWith('.xlsx')) {
            setErrorV2('Пожалуйста, загрузите файл формата .xlsx');
            return;
        }

        setUploadingV2(true);
        setErrorV2(null);
        setSuccessMessageV2(null);

        const formData = new FormData();
        formData.append('file', file);

        try {
            const res = await axios.post(`${API_URL}/api/v2/upload`, formData, {
                headers: { 'Content-Type': 'multipart/form-data' },
            });

            const processed = res.data.records || [];

            if (processed.length === 0) {
                setSuccessMessageV2('Новых изменений не обнаружено');
            } else {
                const newCount = processed.filter(r => r.is_new).length;
                const updatedCount = processed.length - newCount;
                setSuccessMessageV2(`Обработано записей: ${processed.length} (новых: ${newCount}, обновлённых: ${updatedCount})`);
                fetchRecordsV2();
                fetchFilesV2();
            }
        } catch (e) {
            setErrorV2(`Ошибка загрузки файла: ${e.response?.data || e.message}`);
        } finally {
            setUploadingV2(false);
            e.target.value = '';
        }
    };

    const clearChangedColumnsV2 = async () => {
        if (!window.confirm("Вы уверены, что хотите очистить все значения в колонке ChangedColumns?")) return;

        try {
            const response = await fetch(`${API_URL}/api/v2/clear-changed-columns`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
            });
            const data = await response.json();
            alert(data.message || "ChangedColumns очищены!");
            fetchRecordsV2();
            fetchFilesV2();
        } catch (err) {
            alert("Ошибка: " + err.message);
        }
    };

    const deleteDatabaseV2 = async () => {
        if (!window.confirm("ВНИМАНИЕ! Это удалит ВСЕ данные из базы навсегда. Вы уверены?")) return;

        const secondConfirm = window.prompt("Для подтверждения введите слово: УДАЛИТЬ");
        if (secondConfirm !== "УДАЛИТЬ") {
            alert("Операция отменена");
            return;
        }

        try {
            const response = await fetch(`${API_URL}/api/v2/delete-all-records`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ confirm: "delete" }),
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`Ошибка сервера: ${response.status} — ${errorText}`);
            }

            const data = await response.json();
            alert(data.message || "Все записи успешно удалены!");
            setRecordsV2([]);
            setFilesV2([]);
            setSuccessMessageV2("База данных очищена");
        } catch (err) {
            console.error(err);
            alert("Ошибка при удалении: " + err.message);
        }
    };

    const exportExcelV2 = async () => {
        try {
            const response = await fetch(`${API_URL}/api/v2/export`);
            if (!response.ok) throw new Error('Ошибка экспорта');

            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'leasing_records_v2.xlsx';
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);
        } catch (err) {
            alert("Ошибка экспорта: " + err.message);
        }
    };

    const isColumnChanged = (record, column) => record.changed_columns?.includes(column);
    const getCellClass = (record, column) => {
        if (record.is_new) return 'new-row';
        if (isColumnChanged(record, column)) return 'changed-cell';
        return '';
    };

    const isColumnChangedV2 = (record, column) => record.changed_columns?.includes(column);
    const getCellClassV2 = (record, column) => {
        if (record.is_new) return 'new-row';
        if (isColumnChangedV2(record, column)) return 'changed-cell';
        return '';
    };

    const panes = [
        {
            menuItem: 'Вкладка 1',
            render: () => (
                <Tab.Pane>
                    <Segment>
                        <Header as="h3">Загрузка Excel</Header>
                        <input
                            type="file"
                            accept=".xlsx"
                            onChange={handleFileUpload}
                            disabled={uploading}
                            id="file-upload"
                            style={{ display: 'none' }}
                        />
                        <label htmlFor="file-upload">
                            <Button as="span" primary loading={uploading} disabled={uploading} icon labelPosition="left">
                                <Icon name="upload" />
                                {uploading ? 'Загрузка...' : 'Выбрать файл'}
                            </Button>
                        </label>

                        {error && (
                            <Message negative>
                                <Message.Header>Ошибка</Message.Header>
                                {error}
                            </Message>
                        )}

                        {successMessage && (
                            <Message positive>
                                <Message.Header>Успешно</Message.Header>
                                {successMessage}
                            </Message>
                        )}

                        {files.length > 0 && (
                            <div style={{ marginBottom: 16 }}>
                                <Header as="h4">Загруженные файлы:</Header>
                                <ul>
                                    {files.map(f => (
                                        <li key={f}>{f}</li>
                                    ))}
                                </ul>
                            </div>
                        )}
                    </Segment>

                    <div style={{ margin: '20px', padding: '10px', border: '1px solid red' }}>
                        <h3>Админские действия</h3>
                        <button onClick={clearChangedColumns} style={{ marginRight: '10px', padding: '10px', background: '#ff9800' }}>
                            Очистить ChangedColumns
                        </button>
                        <button onClick={deleteDatabase} style={{ padding: '10px', background: '#f44336', color: 'white' }}>
                            Удалить всю базу
                        </button>
                    </div>

                    <Segment>
                        <Header as="h3">
                            Таблица данных
                            <Button floated="right" size="small" icon labelPosition="left" onClick={fetchRecords} loading={loading}>
                                <Icon name="refresh" />
                                Обновить
                            </Button>
                        </Header>

                        {records.length === 0 ? (
                            <Message info>
                                <Message.Header>Нет данных</Message.Header>
                                Загрузите Excel файл
                            </Message>
                        ) : (
                            <div className="table-wrapper">
                                <Table celled striped className="full-width-table">
                                    <Table.Header>
                                        <Table.Row>
                                            <Table.HeaderCell>Предмет</Table.HeaderCell>
                                            <Table.HeaderCell>Местонахождение</Table.HeaderCell>
                                            <Table.HeaderCell>Тип предмета</Table.HeaderCell>
                                            <Table.HeaderCell>Тип ТС</Table.HeaderCell>
                                            <Table.HeaderCell>VIN</Table.HeaderCell>
                                            <Table.HeaderCell>Год</Table.HeaderCell>
                                            <Table.HeaderCell>Пробег</Table.HeaderCell>
                                            <Table.HeaderCell>Дней в продаже</Table.HeaderCell>
                                            <Table.HeaderCell>Цена</Table.HeaderCell>
                                            <Table.HeaderCell>Разница</Table.HeaderCell>
                                            <Table.HeaderCell>Статус</Table.HeaderCell>
                                            <Table.HeaderCell>Фото</Table.HeaderCell>
                                        </Table.Row>
                                    </Table.Header>

                                    <Table.Body>
                                        {records.map(r => (
                                            <Table.Row key={r.id} className={r.is_new ? 'new-row' : ''}>
                                                <Table.Cell className={getCellClass(r, 'subject')}>{r.subject}</Table.Cell>
                                                <Table.Cell className={getCellClass(r, 'location')}>{r.location}</Table.Cell>
                                                <Table.Cell className={getCellClass(r, 'subject_type')}>{r.subject_type}</Table.Cell>
                                                <Table.Cell className={getCellClass(r, 'vehicle_type')}>{r.vehicle_type}</Table.Cell>
                                                <Table.Cell><strong>{r.vin}</strong></Table.Cell>
                                                <Table.Cell className={getCellClass(r, 'year')}>{r.year}</Table.Cell>
                                                <Table.Cell className={getCellClass(r, 'mileage')}>{r.mileage}</Table.Cell>
                                                <Table.Cell className={getCellClass(r, 'days_on_sale')}>{r.days_on_sale}</Table.Cell>
                                                <Table.Cell className={getCellClass(r, 'approved_price')}>
                                                    {r.old_price && r.old_price !== r.approved_price && (
                                                        <span style={{ color: 'red', marginRight: '5px' }}>{r.old_price}</span>
                                                    )}
                                                    <span style={{ color: 'green' }}>{r.approved_price}</span>
                                                </Table.Cell>
                                                <Table.Cell>
                                                    <span style={{ color: 'green' }}>
                                                        {parseFloat((r.old_price || "0").replace(/,/g, "")) - parseFloat((r.approved_price || "0").replace(/,/g, ""))}
                                                    </span>
                                                </Table.Cell>
                                                <Table.Cell className={getCellClass(r, 'status')}>{r.status}</Table.Cell>
                                                <Table.Cell>
                                                    {r.photos?.length > 0 ? (
                                                        <Image.Group size="tiny">
                                                            {r.photos.map((p, i) => (
                                                                <Image key={i} src={p} />
                                                            ))}
                                                        </Image.Group>
                                                    ) : (
                                                        <span className="muted">Нет фото</span>
                                                    )}
                                                </Table.Cell>
                                            </Table.Row>
                                        ))}
                                    </Table.Body>
                                </Table>
                            </div>
                        )}
                    </Segment>
                </Tab.Pane>
            ),
        },
        {
            menuItem: 'Вкладка 2',
            render: () => (
                <Tab.Pane>
                    <Segment>
                        <Header as="h3">Загрузка Excel</Header>
                        <input
                            type="file"
                            accept=".xlsx"
                            onChange={handleFileUploadV2}
                            disabled={uploadingV2}
                            id="file-upload-v2"
                            style={{ display: 'none' }}
                        />
                        <label htmlFor="file-upload-v2">
                            <Button as="span" primary loading={uploadingV2} disabled={uploadingV2} icon labelPosition="left">
                                <Icon name="upload" />
                                {uploadingV2 ? 'Загрузка...' : 'Выбрать файл'}
                            </Button>
                        </label>

                        {errorV2 && (
                            <Message negative>
                                <Message.Header>Ошибка</Message.Header>
                                {errorV2}
                            </Message>
                        )}

                        {successMessageV2 && (
                            <Message positive>
                                <Message.Header>Успешно</Message.Header>
                                {successMessageV2}
                            </Message>
                        )}

                        {filesV2.length > 0 && (
                            <div style={{ marginBottom: 16 }}>
                                <Header as="h4">Загруженные файлы:</Header>
                                <ul>
                                    {filesV2.map(f => (
                                        <li key={f}>{f}</li>
                                    ))}
                                </ul>
                            </div>
                        )}
                    </Segment>

                    <div style={{ margin: '20px', padding: '10px', border: '1px solid red' }}>
                        <h3>Админские действия</h3>
                        <button onClick={clearChangedColumnsV2} style={{ marginRight: '10px', padding: '10px', background: '#ff9800' }}>
                            Очистить ChangedColumns
                        </button>
                        <button onClick={deleteDatabaseV2} style={{ marginRight: '10px', padding: '10px', background: '#f44336', color: 'white' }}>
                            Удалить всю базу
                        </button>
                        <button onClick={exportExcelV2} style={{ padding: '10px', background: '#4caf50', color: 'white' }}>
                            Экспорт в Excel
                        </button>
                    </div>

                    <Segment>
                        <Header as="h3">
                            Таблица данных
                            <Button floated="right" size="small" icon labelPosition="left" onClick={fetchRecordsV2} loading={loadingV2}>
                                <Icon name="refresh" />
                                Обновить
                            </Button>
                        </Header>

                        {recordsV2.length === 0 ? (
                            <Message info>
                                <Message.Header>Нет данных</Message.Header>
                                Загрузите Excel файл
                            </Message>
                        ) : (
                            <div className="table-wrapper">
                                <Table celled striped className="full-width-table">
                                    <Table.Header>
                                        <Table.Row>
                                            <Table.HeaderCell>Марка</Table.HeaderCell>
                                            <Table.HeaderCell>Модель</Table.HeaderCell>
                                            <Table.HeaderCell>VIN</Table.HeaderCell>
                                            <Table.HeaderCell>Срок экспозиции(дн.)</Table.HeaderCell>
                                            <Table.HeaderCell>Вид ТС</Table.HeaderCell>
                                            <Table.HeaderCell>Подвид ТС</Table.HeaderCell>
                                            <Table.HeaderCell>Год выпуска</Table.HeaderCell>
                                            <Table.HeaderCell>Пробег</Table.HeaderCell>
                                            <Table.HeaderCell>Город</Table.HeaderCell>
                                            <Table.HeaderCell>Актуальная стоимость</Table.HeaderCell>
                                            <Table.HeaderCell>Старая стоимость</Table.HeaderCell>
                                            <Table.HeaderCell>Разница</Table.HeaderCell>
                                            <Table.HeaderCell>Статус</Table.HeaderCell>
                                            <Table.HeaderCell>Фото</Table.HeaderCell>
                                        </Table.Row>
                                    </Table.Header>

                                    <Table.Body>
                                        {recordsV2.map(r => (
                                            <Table.Row key={r.id} className={r.is_new ? 'new-row' : ''}>
                                                <Table.Cell className={getCellClassV2(r, 'brand')}>{r.brand}</Table.Cell>
                                                <Table.Cell className={getCellClassV2(r, 'model')}>{r.model}</Table.Cell>
                                                <Table.Cell><strong>{r.vin}</strong></Table.Cell>
                                                <Table.Cell className={getCellClassV2(r, 'exposure_period')}>{r.exposure_period}</Table.Cell>
                                                <Table.Cell className={getCellClassV2(r, 'vehicle_type')}>{r.vehicle_type}</Table.Cell>
                                                <Table.Cell className={getCellClassV2(r, 'vehicle_subtype')}>{r.vehicle_subtype}</Table.Cell>
                                                <Table.Cell className={getCellClassV2(r, 'year')}>{r.year}</Table.Cell>
                                                <Table.Cell className={getCellClassV2(r, 'mileage')}>{r.mileage}</Table.Cell>
                                                <Table.Cell className={getCellClassV2(r, 'city')}>{r.city}</Table.Cell>
                                                <Table.Cell className={getCellClassV2(r, 'actual_price')}>
                                                    {r.old_price && r.old_price !== r.actual_price && (
                                                        <span style={{ color: 'red', marginRight: '5px' }}>{r.old_price}</span>
                                                    )}
                                                    <span style={{ color: 'green' }}>{r.actual_price}</span>
                                                </Table.Cell>
                                                <Table.Cell>
                                                    {r.old_price && r.old_price !== r.actual_price && (
                                                        <span style={{ color: 'red' }}>{r.old_price}</span>
                                                    )}
                                                </Table.Cell>
                                                <Table.Cell>
                                                    <span style={{ color: 'green' }}>
                                                        {parseFloat((r.old_price || "0").replace(/,/g, "")) - parseFloat((r.actual_price || "0").replace(/,/g, ""))}
                                                    </span>
                                                </Table.Cell>
                                                <Table.Cell className={getCellClassV2(r, 'status')}>{r.status}</Table.Cell>
                                                <Table.Cell>
                                                    {r.photos?.length > 0 ? (
                                                        <Image.Group size="tiny">
                                                            {r.photos.map((p, i) => (
                                                                <Image key={i} src={p} />
                                                            ))}
                                                        </Image.Group>
                                                    ) : (
                                                        <span className="muted">Нет фото</span>
                                                    )}
                                                </Table.Cell>
                                            </Table.Row>
                                        ))}
                                    </Table.Body>
                                </Table>
                            </div>
                        )}
                    </Segment>
                </Tab.Pane>
            ),
        },
    ];

    return (
        <Container fluid className="page-container">
            <Header as="h1" textAlign="center">
                <Icon name="shipping fast" />
                Система управления лизинговыми данными
            </Header>

            <Tab panes={panes} activeIndex={activeTab} onTabChange={(e, { activeIndex }) => setActiveTab(activeIndex)} />
        </Container>
    );
}

export default App;