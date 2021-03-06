/// <reference path="../References.d.ts"/>
import * as React from 'react';
import * as ServiceTypes from '../types/ServiceTypes';

interface Props {
	server: ServiceTypes.Server;
	onChange: (state: ServiceTypes.Server) => void;
	onRemove: () => void;
}

const css = {
	group: {
		width: '100%',
		maxWidth: '310px',
		marginTop: '5px',
	} as React.CSSProperties,
	protocol: {
		flex: '0 1 auto',
	} as React.CSSProperties,
	hostname: {
		width: '100%',
	} as React.CSSProperties,
	hostnameBox: {
		flex: '1',
	} as React.CSSProperties,
	port: {
		flex: '0 1 auto',
		width: '52px',
		borderRadius: '0 3px 3px 0',
	} as React.CSSProperties,
};

export default class ServiceServer extends React.Component<Props, {}> {
	clone(): ServiceTypes.Server {
		return {
			...this.props.server,
		};
	}

	render(): JSX.Element {
		let server = this.props.server;

		return <div className="pt-control-group" style={css.group}>
			<div className="pt-select" style={css.protocol}>
				<select
					value={server.protocol}
					onChange={(evt): void => {
						let state = this.clone();
						state.protocol = evt.target.value;
						this.props.onChange(state);
					}}
				>
					<option value="http">HTTP</option>
					<option value="https">HTTPS</option>
				</select>
			</div>
			<div style={css.hostnameBox}>
				<input
					className="pt-input"
					style={css.hostname}
					type="text"
					autoCapitalize="off"
					spellCheck={false}
					placeholder="Hostname"
					value={server.hostname || ''}
					onChange={(evt): void => {
						let state = this.clone();
						state.hostname = evt.target.value;
						this.props.onChange(state);
					}}
				/>
			</div>
			<input
				className="pt-input"
				style={css.port}
				type="text"
				autoCapitalize="off"
				spellCheck={false}
				placeholder="Port"
				value={server.port}
				onChange={(evt): void => {
					let state = this.clone();
					state.port = parseInt(evt.target.value, 10);
					this.props.onChange(state);
				}}
			/>
			<button
				className="pt-button pt-minimal pt-intent-danger pt-icon-remove"
				onClick={(): void => {
					this.props.onRemove();
				}}
			/>
		</div>;
	}
}
